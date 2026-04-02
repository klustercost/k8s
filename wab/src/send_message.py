import requests
import os
from . import logger

ACCESS_TOKEN = os.getenv("ACCESS_TOKEN")
headers = {
    "Authorization": f"Bearer {ACCESS_TOKEN}",
    "Content-Type": "application/json"
}

def send_templated_message(to, template, phone_number_id, message, context):
    url = f"https://graph.facebook.com/v22.0/{phone_number_id}/messages"
    payload = {
        "messaging_product": "whatsapp",
        "to": to,
        "type": "template",
        "template": {
            "name": template,
            "language": {
            "code": "en"
            },
            "components": [
                {
                "type": "body",
                "parameters": [
                    {
                    "type": "text",
                    "text": message
                    }
                ]
                },
                {
                "type": "button",
                "sub_type":"url",
                "index":0,
                "parameters": [
                    {
                    "type": "text",
                    "text": context
                    }
                ]
                }        
            ]
        }
    }
    response = requests.post(url, json=payload, headers=headers)
    if response.status_code != 200:
        logger.log.error(f"Failed to send message to {to}. Response: {response.status_code} {response.text}")
    else:
        logger.log.info(f"Message sent to {to}.")

def send_message(to, message, phone_number_id):
    url = f"https://graph.facebook.com/v22.0/{phone_number_id}/messages"
    payload = {
        "messaging_product": "whatsapp",
        "to": to,
        "text": {"body": message}
    }
    response = requests.post(url, json=payload, headers=headers)
    if response.status_code != 200:
        logger.log.error(f"Failed to send message to {to}. Response: {response.status_code} {response.text}")
    else:
        logger.log.info(f"Message sent to {to}.")