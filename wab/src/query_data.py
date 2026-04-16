import requests
import os
import json

_data_service_address = os.getenv('MCP_CLIENT_ADDRESS')

def query(phone_number_id:str,message:str) -> str:
    response = requests.post(_data_service_address+"/ask",json={ "question":message, "user_id": phone_number_id})
    return "internal error" if response.status_code != 200 else json.loads(response.text)["answer"]
