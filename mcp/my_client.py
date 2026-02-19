import os
import asyncio
import logging
from fastmcp import Client

logging.getLogger("httpx").setLevel(logging.WARNING)
logging.getLogger("httpcore").setLevel(logging.WARNING)
logging.getLogger("fastmcp").setLevel(logging.WARNING)

MCP_SERVER_URL = os.getenv("MCP_SERVER_URL", "http://localhost:8000/mcp")
client = Client(MCP_SERVER_URL)


async def ask(question: str):
    async with client:
        result = await client.call_tool("ask_db", {"question": question})
        if result.is_error:
            print(f"Error: {result.data}")
        else:
            print(result.data)


def main():
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
            print(f"Error: {e}")
        print()


if __name__ == "__main__":
    main()
