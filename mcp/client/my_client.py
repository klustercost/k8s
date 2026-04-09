import os
import logging

import uvicorn
from fastapi import FastAPI
from fastapi.responses import JSONResponse
from fastmcp import Client
from pydantic import BaseModel, field_validator
from dotenv import load_dotenv

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


class AskRequest(BaseModel):
    question: str

    @field_validator("question")
    @classmethod
    def question_not_blank(cls, v: str) -> str:
        stripped = v.strip()
        if not stripped:
            raise ValueError("question must not be blank")
        return stripped


async def ask(question: str) -> dict:
    """Forward a question to the MCP server and return the result."""
    async with Client(MCP_SERVER_URL) as mcp:
        result = await mcp.call_tool("ask_db", {"question": question})
        if result.is_error:
            log.error(result.data)
            return {"error": str(result.data)}
        log.info(result.data)
        return {"answer": result.data}


@app.post("/ask")
async def post_ask(body: AskRequest):
    log.info("──── New question received ────")
    log.info("User question: %s", body.question)

    try:
        result = await ask(body.question)
    except Exception:
        log.exception("Failed to process question")
        return JSONResponse(status_code=500, content={"error": "Internal server error"})

    log.info("──── Question complete ────")
    if "error" in result:
        return JSONResponse(status_code=500, content=result)
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
