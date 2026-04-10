import os
from dotenv import load_dotenv

load_dotenv()

class Config:
    CLIENT_ID = os.environ.get("CLIENT_ID", "")
    BOT_TYPE = os.environ.get("BOT_TYPE", "")
    MCP_CLIENT_ADDRESS = os.getenv("MCP_CLIENT_ADDRESS", "")
