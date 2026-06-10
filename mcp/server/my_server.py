import json
import logging
import os

from dotenv import load_dotenv
from fastmcp import FastMCP, Context

from db_reader import process_question
from ddl_generator import PG_SCHEMA, process_jsonata_ddl_request

load_dotenv(override=False)

LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger("mcp-server")

mcp = FastMCP("My MCP Server")


@mcp.tool
async def ask_db(question: str, response_id: str | None, ctx: Context) -> str:
    """Ask a natural-language question about the PostgreSQL database.

    The question is converted to SQL via OpenAI, executed, and the results
    are returned as a JSON object with two keys:
      - "raw":     the structured query results (list of row objects)
      - "natural": a human-readable, conversational answer (or null on failure)
    """
    log.info("──── New question received ────")
    log.info(f"User question: {question}")
    log.info(f"Previous response ID: {response_id}")

    result = process_question(question, response_id)

    if result.get("status") == "success":
        log.info(f"Returning {len(result['raw'])} row(s) to client")
        log.debug(f"Full result payload:\n{result}")
        log.info("──── Question complete ────")

    return json.dumps(result, indent=2, default=str)


@mcp.tool
async def translate_jsonata_to_psql(
    jsonata: str,
    table_name: str,
    type_name: str | None = None,
    schema: str | None = None,
    index_columns: list[str] | None = None,
    ctx: Context = None,
) -> str:
    """Translate a JSONata object expression into PostgreSQL DDL.

    This tool only returns SQL text. It never executes the generated DDL.
    """
    log.info("---- New JSONata DDL request received ----")
    log.info(f"Target table: {table_name}")
    log.info(f"Requested type: {type_name}")
    log.info(f"Requested schema: {schema}")
    try:
        result = process_jsonata_ddl_request(
            jsonata,
            table_name,
            type_name=type_name,
            schema=schema,
            index_columns=index_columns,
        )
        log.info("---- JSONata DDL request complete ----")
        return json.dumps(result, indent=2)
    except Exception as e:
        log.error("translate_jsonata_to_psql failed: %s", e)
        return json.dumps({
            "status": "error",
            "error": str(e),
            "ddl": None,
            "table_name": table_name,
            "type_name": type_name,
            "schema": schema or PG_SCHEMA,
            "warnings": [],
        }, indent=2)


if __name__ == "__main__":
    mcp.run(
        transport="streamable-http",
        host=os.getenv("MCP_HOST", "0.0.0.0"),
        port=int(os.getenv("MCP_PORT", "8000")),
        path="/mcp",
    )
