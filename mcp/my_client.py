import asyncio
from fastmcp import Client

client = Client("http://localhost:8000/mcp")


async def call_tool(name: str, args: dict):
    async with client:
        result = await client.call_tool(name, args)
        print(result)


# Example: ask a natural-language question about the database
asyncio.run(call_tool("ask_db", {"question": "Which pod consumed the most CPU in the last 1 hour?"}))
