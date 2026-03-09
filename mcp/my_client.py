import os
import json
import asyncio
import logging
from http.server import HTTPServer, BaseHTTPRequestHandler
from fastmcp import Client

LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger("mcp-client")

logging.getLogger("httpx").setLevel(logging.WARNING)
logging.getLogger("httpcore").setLevel(logging.WARNING)
logging.getLogger("fastmcp").setLevel(logging.WARNING)

# --- Configuration from environment ---
MCP_SERVER_URL = os.getenv("MCP_SERVER_URL", "http://localhost:8000/mcp")
MCP_CLIENT_HOST = os.getenv("MCP_CLIENT_HOST", "0.0.0.0")
MCP_CLIENT_PORT = int(os.getenv("MCP_CLIENT_PORT", "8080"))


async def ask(question: str) -> dict:
    """Forward a question to the MCP server and return the result."""
    async with Client(MCP_SERVER_URL) as mcp:
        result = await mcp.call_tool("ask_db", {"question": question})
        if result.is_error:
            log.error(result.data)
            return {"error": str(result.data)}
        log.info(result.data)
        return {"answer": result.data}


class RequestHandler(BaseHTTPRequestHandler):
    """HTTP request handler that exposes the /ask endpoint."""

    def do_POST(self):
        if self.path != "/ask":
            self._send_json(404, {"error": "Not found"})
            return

        try:
            content_length = int(self.headers.get("Content-Length", 0))
        except (ValueError, TypeError):
            self._send_json(400, {"error": "Invalid Content-Length header"})
            return
        if content_length == 0:
            self._send_json(400, {"error": "Empty request body"})
            return

        try:
            body = json.loads(self.rfile.read(content_length))
        except json.JSONDecodeError:
            self._send_json(400, {"error": "Invalid JSON"})
            return

        if not isinstance(body, dict):
            self._send_json(400, {"error": "Request body must be a JSON object"})
            return

        question = body.get("question")
        if not isinstance(question, str) or not question.strip():
            self._send_json(400, {"error": "Missing or empty 'question' field"})
            return
        question = question.strip()

        log.info("──── New question received ────")
        log.info("User question: %s", question)

        try:
            result = asyncio.run(ask(question))
        except Exception:
            log.exception("Failed to process question")
            self._send_json(500, {"error": "Internal server error"})
            return

        status = 500 if "error" in result else 200
        self._send_json(status, result)
        log.info("──── Question complete ────")

    def do_GET(self):
        if self.path == "/healthz":
            self._send_json(200, {"status": "ok"})
            return
        self._send_json(404, {"error": "Not found"})

    def _send_json(self, status_code: int, data: dict):
        payload = json.dumps(data, indent=2, default=str).encode("utf-8")
        self.send_response(status_code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)


def main():
    server = HTTPServer((MCP_CLIENT_HOST, MCP_CLIENT_PORT), RequestHandler)
    log.info("MCP client HTTP server listening on %s:%d", MCP_CLIENT_HOST, MCP_CLIENT_PORT)
    log.info("MCP server URL: %s", MCP_SERVER_URL)
    log.info("POST /ask   — send a question to the AI")
    log.info("GET  /healthz — health check")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        log.info("Shutting down")
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
