from . import logger
from .send_message import send_message, send_templated_message
from .query_data import query
import os
import json

TEMPLATE = os.environ.get("TEMPLATE")
TEMPLATE_CONTEXT = os.environ.get("TEMPLATE_CONTEXT")

def handle_message(message, phone_number_id):
    sender_id = message["from"]
    response = json.loads(query(phone_number_id,message["text"]["body"]))
    natural_response = response["natural"]
    logger.log.info(f"Handling from ${sender_id} request ${message['text']['body']} with answer ${natural_response}")
    send_message(sender_id,natural_response,phone_number_id)
    if ( response["status"] == "success" ):
        send_templated_message(sender_id, TEMPLATE, phone_number_id, "this request", TEMPLATE_CONTEXT)
