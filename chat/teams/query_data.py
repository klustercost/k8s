import requests
import json

from config import Config

def query(config: Config, message: str) -> str:
    response = requests.post(config.MCP_CLIENT_ADDRESS+"/ask",json={ "question":message })
    if response.status_code != 200:
        return '{"internal error"}'
    else:
        return json.loads(response.text)["answer"]