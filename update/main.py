import logging
import os
import psycopg2
from requests import get
from requests.exceptions import  InvalidSchema
import json
import time
import signal
import sys
from dotenv import load_dotenv


load_dotenv(override=False)

class operate_db:
    __cache = {}

    def __init__(self) -> None:

        try:
            self.price_uri = os.getenv('price_uri')

            self.connection = psycopg2.connect(
                host=os.getenv('host'),
                database=os.getenv('database'),
                user=os.getenv('user'),
                password=os.getenv('password'),
                port=os.getenv('port')
            )
        except KeyError as Ex:
            raise Exception(f"missing an environment variable: {Ex.args[0]}")
        
    def get_work_items(self) -> None:
        logging.debug(f'Checking for nodes with no price')
        try:
            with self.connection.cursor() as cursor:
                cursor.execute("SELECT idx, labels FROM klustercost.tbl_nodes WHERE price_per_hour IS NULL AND labels IS NOT NULL")
                row = cursor.fetchone()
                while row is not None:
                    price = self.price_from_labels(row[1])
                    if price is not None:
                        self.set_work_item(row[0], price)
                    row = cursor.fetchone()
            self.connection.commit()
        except psycopg2.DatabaseError as error:
            logging.error(error)

    def price_from_labels(self, labels) -> float:
        if not labels in self.__cache:       
            tokens = labels.split(',')
            node_data = {}
            for token in tokens:
                key_val = token.split('=')
                if len(key_val) >= 2:
                    node_data[key_val[0]] = key_val[1]
            val = self.query_provider(node_data)
            if val:
                self.__cache[labels] = val
            else:
                return val

        return self.__cache[labels]

    def query_provider(self,node_data) -> float:
        try:
            response = get(
                f"http://{self.price_uri}/get",
                params={
                    "region":node_data['topology.kubernetes.io/region'],
                    "sku":node_data['node.kubernetes.io/instance-type'],
                    "os":node_data['kubernetes.io/os']
            })
            if response.status_code == 200:
                return json.loads(response.text)[0][1]
        except (KeyError, IndexError, InvalidSchema) as Ex:
            logging.error(
                "Error: no data for: region=%s&sku=%s&os=%s (%s)",
                node_data.get('topology.kubernetes.io/region', 'MISSING'),
                node_data.get('node.kubernetes.io/instance-type', 'MISSING'),
                node_data.get('kubernetes.io/os', 'MISSING'),
                Ex,
            )

    def set_work_item(self, idx, price) -> None:
        logging.info(f'Setting at {idx} cost of {price}')        
        try:
            self.connection.cursor().execute(
                "UPDATE klustercost.tbl_nodes SET price_per_hour = %s WHERE idx = %s",
                (price, idx),
            )
        except psycopg2.DatabaseError as error:
            logging.error(error)        

def signal_handler(sig, frame):
    logging.info('Leaving')
    sys.exit(0)

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    logging.info('This is the klustercost price updater')

    signal.signal(signal.SIGINT, signal_handler)

    operate_db = operate_db()
    while True:
        operate_db.get_work_items()
        time.sleep(10)