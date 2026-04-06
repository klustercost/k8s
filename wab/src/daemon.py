from . import logger
from .send_message import send_message, send_templated_message
from .query_data import query
import os
import json

TEMPLATE = os.environ.get("TEMPLATE")

crt_context="_9dG8Nkit9?ctid=00a13d3a-b034-4fc8-b3b0-b1b37777fd93&pbi_source=linkShare&bookmarkGuid=c21ef7da-f8a2-4a4f-9869-d748638fae86"

def handle_message(message, phone_number_id):
    sender_id = message["from"]    
    response = query(message["text"]["body"])
    natural_response = json.loads(response)["natural"]
    logger.log.info(f"Handling from ${sender_id} request ${response} with answer ${natural_response}")
    send_message(sender_id,natural_response,phone_number_id)
    send_templated_message(sender_id, TEMPLATE, phone_number_id, "xxx", crt_context)
