import logging
import os
import psycopg2
import urllib.request
import json
import time

class operate_db:
    __cache = {}

    def __init__(self) -> None:
        try:
            self.price_uri = os.environ['price_uri']

            self.connection = psycopg2.connect(
                host=os.environ['host'],
                database=os.environ['database'],
                user=os.environ['user'],
                password=os.environ['password']
            )
        except KeyError as Ex:
            raise Exception(f"missing an environment variable: {Ex.args[0]}")
        
    def get_work_items(self) -> None:
        logging.debug(f'Checking for nodes with no price')
        try:
            with self.connection.cursor() as cursor:
                cursor.execute("SELECT idx, labels FROM klustercost.tbl_nodes where price_per_hour is null")                
                row = cursor.fetchone()
                while row is not None:
                    self.set_work_item(row[0],self.price_from_labels(row[1]))
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
                node_data[key_val[0]] = key_val[1]
            val = self.query_provider(node_data)
            self.__cache[labels] = val

        return self.__cache[labels]

    def query_provider(self,node_data) -> float:
        request = f"http://{self.price_uri}/get?region={node_data["topology.kubernetes.io/region"]}&sku={node_data["node.kubernetes.io/instance-type"]}&os={node_data["kubernetes.io/os"]}"
        logging.info(request)
        var = json.loads(urllib.request.urlopen(request).read().decode("utf-8"))
        return var[0][1]

    def set_work_item(self, idx, price) -> None:
        logging.debug(f'Setting at {idx} cost of {price}')        
        try:
            self.connection.cursor().execute(f"UPDATE klustercost.tbl_nodes set price_per_hour = {price} where idx = {idx}")
        except psycopg2.DatabaseError as error:
            logging.error(error)        

if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    logging.info('This is the klustercost price updater')

    operate_db = operate_db()
    while True:
        operate_db.get_work_items()
        time.sleep(10)