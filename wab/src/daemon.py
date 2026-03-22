from . import logger
from .send_message import send_message
from .query_data import query



def handle_message(message, phone_number_id):
    sender_id = message["from"]    
    response = query(message["text"]["body"])
    logger.log.info("Handling from %s request %s ",sender_id,response)
    send_message(sender_id, response, phone_number_id)