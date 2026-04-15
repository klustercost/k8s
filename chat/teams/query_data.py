import requests
import json

def query(url: str, user_id: str, message: str) -> str:
    response = requests.post(f"{url}/ask", json={"user_id": user_id,"question": message})
    if response.status_code != 200:
        return '{"internal error"}'
    else:
        return json.loads(response.text)["answer"]