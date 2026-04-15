import os
import logging
from json import loads, JSONDecodeError

import uvicorn
from fastapi import FastAPI, Request, HTTPException
from fastmcp import Client
from dotenv import load_dotenv

from active_users import get_last_request, set_last_request

load_dotenv(override=False)

LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger("mcp-client")

# Suppress verbose per-request logs from transitive dependencies and uvicorn's
# built-in access logger so only our own application-level logs are visible.
for noisy in ("httpx", "httpcore", "fastmcp", "uvicorn.access"):
    logging.getLogger(noisy).setLevel(logging.WARNING)

# --- Configuration from environment ---
MCP_SERVER_URL = os.getenv("MCP_SERVER_URL", "http://localhost:8000/mcp")
MCP_CLIENT_HOST = os.getenv("MCP_CLIENT_HOST", "0.0.0.0")
MCP_CLIENT_PORT = int(os.getenv("MCP_CLIENT_PORT", "8080"))

app = FastAPI(title="MCP Client", docs_url=None, redoc_url=None)


MCP_TIMEOUT = int(os.getenv("MCP_TIMEOUT", "120"))

async def ask(question: str, user_id: str) -> dict:
    """Forward a question to the MCP server using a per-user session."""
    async with Client(MCP_SERVER_URL) as mcp:
        result = await mcp.call_tool(
            "ask_db",
            {"question": question, "response_id": get_last_request(user_id)},
            raise_on_error=False,
            timeout=MCP_TIMEOUT,
        )
        if result.is_error:
            text = result.data or result.content[0].text
            log.error(text)
            return {"error": str(text)}
        payload = result.data if result.data is not None else result.structured_content
        try:
            set_last_request(user_id, loads(payload)["response_id"])
        except Exception as e:
            log.error(f"Failed to parse response_id from MCP response: {e}")
        log.info(payload)
        return {"answer": payload}


async def query_from_body(request: Request):
    try:    
        body = await request.json()
        if not isinstance(body, dict):
            raise HTTPException(status_code=400, detail={"error": "Expected JSON object"})
        question = body.get("question")
        user_id = body.get("user_id")
        if not isinstance(question, str) or not question.strip():
            raise HTTPException(status_code=400, detail={"error": "Missing or empty 'question' field"})
        log.info("──── New question received ────")
        log.info(f"User: {user_id} | Question: {question}")
        return question, user_id
    except JSONDecodeError:
        raise HTTPException(status_code=400, detail={"error": "Expect json body"})

@app.post("/ask")
async def post_ask(request: Request):
    question, user_id = await query_from_body(request)

    try:
        result = await ask(question, user_id)
    except Exception as e:
        log.exception(f"Failed to process question {e}")
        raise HTTPException(status_code=500, detail={"error": "Internal server error"})

    log.info("──── Question complete ────")
    if "error" in result:
        raise HTTPException(status_code=500, detail=result)
    return result


@app.get("/healthz")
async def healthz():
    return {"status": "ok"}


if __name__ == "__main__":
    log.info("MCP server URL: %s", MCP_SERVER_URL)
    uvicorn.run(
        app,
        host=MCP_CLIENT_HOST,
        port=MCP_CLIENT_PORT,
    )
