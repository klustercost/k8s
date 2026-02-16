import asyncio
from fastmcp import Client

client = Client("http://localhost:8000/mcp")


async def ask(question: str):
    async with client:
        result = await client.call_tool("ask_db", {"question": question})
        print(result)


def main():
    print("Connected to MCP server at http://localhost:8000/mcp")
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
