import requests
import os
import json

data_service_address = os.getenv('MCP_CLIENT_ADDRESS')

def query(message:str) -> str:
    response = requests.post(data_service_address+"/ask",json={ "question":message })
    if response.status_code != 200:
        return "internal error"
    else:
        return json.loads(response.text)["answer"]