import os
import asyncio
import logging
import time
from fastmcp import Client

LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger("mcp-client")

MCP_SERVER_URL = os.getenv("MCP_SERVER_URL", "http://localhost:8000/mcp")
client = Client(MCP_SERVER_URL)


async def ask(question: str):
    log.info("Sending question to MCP server: %s", question)
    t0 = time.perf_counter()
    async with client:
        result = await client.call_tool("ask_db", {"question": question})
        elapsed = time.perf_counter() - t0
        if result.is_error:
            log.error("Server returned error after %.2fs: %s", elapsed, result.data)
            print(f"Error: {result.data}")
        else:
            log.info("Response received in %.2fs", elapsed)
            log.debug("Response payload: %s", result.data)
            print(result.data)


def main():
    log.info("MCP client starting — server URL: %s", MCP_SERVER_URL)
    print(f"Connected to MCP server at {MCP_SERVER_URL}")
    print("Type your question and press Enter. Type 'exit' to quit.\n")
    while True:
        try:
            question = input("Question: ").strip()
        except (KeyboardInterrupt, EOFError):
            print("\nBye!")
            break
        if not question:
            continue
        if question.lower() in ("exit", "quit"):
            print("Bye!")
            break
        try:
            asyncio.run(ask(question))
        except Exception as e:
            log.error("Unhandled error: %s", e)
            print(f"Error: {e}")
        print()


if __name__ == "__main__":
    main()
