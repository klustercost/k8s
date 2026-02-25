import os
import asyncio
import logging
from fastmcp import Client

LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO").upper()
logging.basicConfig(level=getattr(logging, LOG_LEVEL, logging.INFO), format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger(__name__)

logging.getLogger("httpx").setLevel(logging.WARNING)
logging.getLogger("httpcore").setLevel(logging.WARNING)
logging.getLogger("fastmcp").setLevel(logging.WARNING)

MCP_SERVER_URL = os.getenv("MCP_SERVER_URL", "http://localhost:8000/mcp")
client = Client(MCP_SERVER_URL)


async def ask(question: str):
    async with client:
        result = await client.call_tool("ask_db", {"question": question})
        if result.is_error:
            log.error(result.data)
        else:
            log.info(result.data)


def main():
    log.info("Connected to MCP server at %s", MCP_SERVER_URL)
    log.info("Type your question and press Enter. Type 'exit' to quit.")
    while True:
        try:
            question = input("Question: ").strip()
        except (KeyboardInterrupt, EOFError):
            log.info("Bye!")
            break
        if not question:
            continue
        if question.lower() in ("exit", "quit"):
            log.info("Bye!")
            break
        try:
            asyncio.run(ask(question))
        except Exception as e:
            log.exception("Failed to process question")


if __name__ == "__main__":
    main()
