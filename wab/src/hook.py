from fastapi import FastAPI, Request, HTTPException, Query
import os
from .daemon import handle_message

app = FastAPI()
VERIFY_TOKEN = os.getenv("VERIFY_TOKEN")

# Health check
@app.get("/")
async def home():
    return {"message": "API is up"}

# Webhook verification
@app.get("/webhook")
async def verify_webhook(
    hub_mode: str = Query(None, alias="hub.mode"),
    hub_challenge: str = Query(None, alias="hub.challenge"),
    hub_verify_token: str = Query(None, alias="hub.verify_token")
):
    if hub_mode == "subscribe" and hub_verify_token == VERIFY_TOKEN:
        return int(hub_challenge)
    raise HTTPException(status_code=403, detail="Forbidden")

# Handle incoming messages
@app.post("/webhook")
async def handle_webhook(request: Request):
    data = await request.json()
    #print("Received webhook:", data)

    if data:
        for entry in data.get("entry", []):
            for change in entry.get("changes", []):
                value = change.get("value", {})
                phone_number_id = value.get("metadata", {}).get("phone_number_id")
                message_data = value.get("messages", [])
                for message in message_data:
                    handle_message(message, phone_number_id)

    return {"status": "EVENT_RECEIVED"}