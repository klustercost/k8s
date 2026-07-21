import logging
import psycopg2
from config import config

class call_db:
    def __init__(self) -> None:        
        try:
            self._connection = psycopg2.connect(
                host=config.db_host,
                database=config.db_database,
                user=config.db_user,
                password=config.db_password,
                port=config.db_port
            )
        except psycopg2.DatabaseError as error:
            logging.error(error)
            raise error

    def call(self, query: str, params: tuple = None) -> list:
        logging.debug(f'Calling db with query: {query} and params: {params}')
        try:
            with self._connection as connection:
                with connection.cursor() as cursor:
                    cursor.execute(query, params)
                    if cursor.rowcount < 0:
                        return []
                    result = cursor.fetchall()
                    return result
        except psycopg2.DatabaseError as error:
            logging.error(error)
            raise error
