import os
import json
import logging
import re
import time
from functools import lru_cache

import psycopg2
from dotenv import load_dotenv
from openai import OpenAI
from fastmcp import FastMCP, Context
from dotenv import load_dotenv
from answer_formatter import format_answer

load_dotenv(override=False)

LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger("mcp-server")

# System prompt file
PROMPT_FILE = os.path.join(os.path.dirname(__file__), "system_prompt.txt")
with open(PROMPT_FILE, encoding="utf-8") as f:
    SYSTEM_PROMPT_TEMPLATE = f.read()

JSONATA_DDL_PROMPT_FILE = os.path.join(os.path.dirname(__file__), "jsonata_ddl_prompt.txt")
with open(JSONATA_DDL_PROMPT_FILE, encoding="utf-8") as f:
    JSONATA_DDL_PROMPT_TEMPLATE = f.read()

# --- Configuration from .env ---
OPENAI_API_KEY = os.getenv("OPENAI_API_KEY")
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-5.4-mini")
OPENAI_ANSWER_MODEL = os.getenv("OPENAI_ANSWER_MODEL", OPENAI_MODEL)
PG_HOST = os.getenv("PG_HOST", "localhost")
PG_PORT = int(os.getenv("PG_PORT", "5432"))
PG_USER = os.getenv("PG_USER", "postgres")
PG_PASSWORD = os.getenv("PG_PASSWORD", "")
PG_DATABASE = os.getenv("PG_DATABASE", "klustercost")
PG_SCHEMA = os.getenv("PG_SCHEMA", "klustercost")

openai_client = OpenAI(api_key=OPENAI_API_KEY)
mcp = FastMCP("My MCP Server")

IDENTIFIER_RE = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")
DISALLOWED_DDL_RE = re.compile(
    r"\b(DROP|DELETE|UPDATE|INSERT|ALTER|TRUNCATE|CALL|COPY|DO|GRANT|REVOKE|EXECUTE|CREATE\s+SCHEMA)\b",
    re.IGNORECASE,
)


def get_pg_connection():
    return psycopg2.connect(
        host=PG_HOST,
        port=PG_PORT,
        user=PG_USER,
        password=PG_PASSWORD,
        dbname=PG_DATABASE,
    )

# cache schema info
# lru_cache stands for Least Recently Used Cache. 
# It's a decorator from functools that remembers the result of a function call so it doesn't have to run again.
# maxsize=1 means it only remembers the last result.
@lru_cache(maxsize=1)
def get_schema_info() -> str:
    """Fetch table and column metadata from information_schema."""
    log.debug("Fetching schema metadata for schema=%s", PG_SCHEMA)
    conn = None
    try:
        conn = get_pg_connection()
        with conn.cursor() as cur:
            cur.execute(
                """
                SELECT table_name, column_name, data_type
                FROM information_schema.columns
                WHERE table_schema = %s
                ORDER BY table_name, ordinal_position
                """,
                (PG_SCHEMA,),
            )
            rows = cur.fetchall()

        tables: dict[str, list[str]] = {}
        for table, column, dtype in rows:
            tables.setdefault(table, []).append(f"{column} ({dtype})")

        lines = []
        for table, cols in tables.items():
            lines.append(f"{PG_SCHEMA}.{table}: {', '.join(cols)}")
        schema_text = "\n".join(lines)
        log.debug("Schema info (%d tables): %s", len(tables), schema_text)
        return schema_text
    finally:
        if conn is not None:
            conn.close()


def generate_sql(question: str, schema: str, response_id: str = None) -> str:
    """Ask OpenAI to produce a read-only SQL query for the given question."""
    log.info("Generating SQL via OpenAI (model=%s) …", OPENAI_MODEL)
    t0 = time.perf_counter()

    response = openai_client.responses.create(
        model=OPENAI_MODEL,
        previous_response_id=response_id,
        instructions=SYSTEM_PROMPT_TEMPLATE.format(schema=schema),
        input=question
    )

    elapsed = time.perf_counter() - t0
    content = response.output_text
    if content is None:
        raise ValueError("OpenAI returned an empty response — no SQL was generated")
    sql = content.strip()
    log.info(f"SQL generated in {elapsed:.2f}:\n{sql}")
    return sql,  response.id


def is_valid_identifier(name: str) -> bool:
    return isinstance(name, str) and bool(IDENTIFIER_RE.match(name))


def derive_type_name(table_name: str) -> str:
    base = table_name
    if base.startswith("tbl_"):
        base = base[4:]
    if base.endswith("s"):
        base = base[:-1]
    return f"{base}_type"


def build_jsonata_ddl_prompt(
    jsonata: str,
    table_name: str,
    type_name: str,
    schema: str,
    index_columns: list[str] | None,
) -> str:
    index_text = ", ".join(index_columns) if index_columns else "infer useful lookup indexes"
    return JSONATA_DDL_PROMPT_TEMPLATE.format(
        jsonata=jsonata,
        schema=schema,
        table_name=table_name,
        type_name=type_name,
        index_text=index_text,
    ).strip()


def parse_jsonata_ddl_response(content: str) -> tuple[str, list[str]]:
    try:
        payload = json.loads(content)
    except json.JSONDecodeError as e:
        raise ValueError(f"OpenAI returned invalid JSON: {e}")

    if not isinstance(payload, dict):
        raise ValueError("OpenAI response must be a JSON object")
    ddl = payload.get("ddl")
    warnings = payload.get("warnings", [])
    if not isinstance(ddl, str) or not ddl.strip():
        raise ValueError("OpenAI response is missing a non-empty 'ddl' string")
    if not isinstance(warnings, list) or not all(isinstance(item, str) for item in warnings):
        raise ValueError("OpenAI response 'warnings' must be a list of strings")
    return ddl.strip(), warnings


def sql_name_pattern(name: str) -> str:
    return rf'(?:"{re.escape(name)}"|{re.escape(name)})'


def validate_jsonata_ddl(ddl: str, table_name: str, type_name: str, schema: str) -> None:
    if "--" in ddl or "/*" in ddl or "*/" in ddl:
        raise ValueError("Generated DDL must not contain SQL comments")
    if DISALLOWED_DDL_RE.search(ddl):
        raise ValueError("Generated DDL contains a disallowed SQL statement")
    unquoted_ddl = re.sub(r'"[^"]*"', "", ddl)
    referenced_schemas = {
        match.group(1)
        for match in re.finditer(r"\b([A-Za-z_][A-Za-z0-9_]*)\s*\.\s*([A-Za-z_][A-Za-z0-9_]*)\b", unquoted_ddl)
    }
    if referenced_schemas - {schema}:
        raise ValueError("Generated DDL must not reference schemas other than the requested schema")

    statements = [statement.strip() for statement in ddl.split(";") if statement.strip()]
    if not statements:
        raise ValueError("Generated DDL is empty")

    table_ref = rf"{sql_name_pattern(schema)}\s*\.\s*{sql_name_pattern(table_name)}"
    type_ref = rf"{sql_name_pattern(schema)}\s*\.\s*{sql_name_pattern(type_name)}"
    has_table = False
    has_type = False

    for statement in statements:
        if re.match(r"^CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\b", statement, re.IGNORECASE):
            if not re.search(rf"^CREATE\s+TABLE\s+IF\s+NOT\s+EXISTS\s+{table_ref}\b", statement, re.IGNORECASE):
                raise ValueError("CREATE TABLE must target the requested schema and table")
            has_table = True
        elif re.match(r"^CREATE\s+TYPE\b", statement, re.IGNORECASE):
            if not re.search(rf"^CREATE\s+TYPE\s+{type_ref}\s+AS\b", statement, re.IGNORECASE):
                raise ValueError("CREATE TYPE must target the requested schema and type")
            has_type = True
        elif re.match(r"^CREATE\s+INDEX\s+IF\s+NOT\s+EXISTS\b", statement, re.IGNORECASE):
            if not re.search(rf"\bON\s+{table_ref}\b", statement, re.IGNORECASE):
                raise ValueError("CREATE INDEX must target the requested schema and table")
        else:
            raise ValueError("Generated DDL contains a statement type that is not allowed")

    if not has_table:
        raise ValueError("Generated DDL must include CREATE TABLE IF NOT EXISTS")
    if not has_type:
        raise ValueError("Generated DDL must include CREATE TYPE")


def generate_jsonata_ddl(
    jsonata: str,
    table_name: str,
    type_name: str,
    schema: str,
    index_columns: list[str] | None,
) -> tuple[str, list[str]]:
    """Ask OpenAI to produce PostgreSQL DDL for a JSONata object expression."""
    log.info("Generating JSONata DDL via OpenAI (model=%s) ...", OPENAI_MODEL)
    t0 = time.perf_counter()

    response = openai_client.responses.create(
        model=OPENAI_MODEL,
        instructions=(
            "You generate conservative PostgreSQL DDL from JSONata object expressions. "
            "You return strict JSON only."
        ),
        input=build_jsonata_ddl_prompt(jsonata, table_name, type_name, schema, index_columns),
    )

    elapsed = time.perf_counter() - t0
    content = response.output_text
    if content is None:
        raise ValueError("OpenAI returned an empty response -- no DDL was generated")
    ddl, warnings = parse_jsonata_ddl_response(content.strip())
    validate_jsonata_ddl(ddl, table_name, type_name, schema)
    log.info(f"JSONata DDL generated in {elapsed:.2f}s:\n{ddl}")
    return ddl, warnings


def run_query(sql: str) -> list[dict]:
    """Execute a SELECT query and return rows as list of dicts."""
    log.info(f"Sending query to PostgreSQL ({PG_HOST}:{PG_PORT}/{PG_DATABASE}) …")
    conn = None
    try:
        t0 = time.perf_counter()
        conn = get_pg_connection()
        with conn.cursor() as cur:
            cur.execute(sql)
            if cur.description is None:
                raise ValueError("Query returned no result set — only SELECT statements are supported")
            columns = [desc[0] for desc in cur.description]
            rows = [dict(zip(columns, row)) for row in cur.fetchall()]
        elapsed = time.perf_counter() - t0
        log.info(f"PostgreSQL responded in {elapsed:.2f}s — {len(rows)} row(s) returned")
        log.debug(f"Result columns: {columns}")
        if rows:
            log.debug(f"First row: {rows[0]}")
        return rows
    finally:
        if conn is not None:
            conn.close()


# --- MCP Tools ---

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
    sql = None
    try:
        schema = get_schema_info()
        sql, response_id = generate_sql(question, schema, response_id)
        if sql.strip() == "REFUSE":
            log.warning("Question refused by LLM (off-topic)")
            return json.dumps({
                "raw": "Sorry, I can only answer questions about the Kubernetes cluster database. Please try again with a question about the cluster. For example: Which pods are using the most CPU today?",
                "natural": "Sorry, I can only answer questions about the Kubernetes cluster database. Please try again with a question about the cluster. For example: Which pods are using the most CPU today?",
                "status": "refused"
            }, indent=2)
        rows = run_query(sql)
    except Exception as e:
        sql_info = f"\nGenerated SQL was:\n{sql}" if sql else ""
        log.error("ask_db failed: %s%s", e, sql_info)
        return json.dumps({
            "raw": f"Error: {e}{sql_info}",
            "natural": f"Error: {e}{sql_info}",
            "status": "error"
        }, indent=2)

    natural = None
    try:
        natural = format_answer(question, json.dumps(rows, default=str), openai_client, OPENAI_ANSWER_MODEL)
    except Exception as e:
        log.error("Answer formatting failed: %s", e)

    result = json.dumps({"response_id": response_id, "raw": rows, "natural": natural,"status": "success"}, indent=2, default=str)
    log.info(f"Returning {len(rows)} row(s) to client")
    log.debug(f"Full result payload:\n{result}")
    log.info("──── Question complete ────")

    return result


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
        if not isinstance(jsonata, str) or not jsonata.strip():
            raise ValueError("Missing or empty 'jsonata' field")
        if not jsonata.strip().startswith("{") or not jsonata.strip().endswith("}"):
            raise ValueError("'jsonata' must be a JSONata object expression")
        if not is_valid_identifier(table_name):
            raise ValueError("'table_name' must be a plain PostgreSQL identifier")

        resolved_schema = schema or PG_SCHEMA
        if not is_valid_identifier(resolved_schema):
            raise ValueError("'schema' must be a plain PostgreSQL identifier")

        resolved_type_name = type_name or derive_type_name(table_name)
        if not is_valid_identifier(resolved_type_name):
            raise ValueError("'type_name' must be a plain PostgreSQL identifier")

        if index_columns is not None:
            if not isinstance(index_columns, list) or not all(isinstance(item, str) and item.strip() for item in index_columns):
                raise ValueError("'index_columns' must be a list of non-empty strings")

        ddl, warnings = generate_jsonata_ddl(
            jsonata.strip(),
            table_name,
            resolved_type_name,
            resolved_schema,
            index_columns,
        )
        result = json.dumps({
            "status": "success",
            "ddl": ddl,
            "table_name": table_name,
            "type_name": resolved_type_name,
            "schema": resolved_schema,
            "warnings": warnings,
        }, indent=2)
        log.info("---- JSONata DDL request complete ----")
        return result
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
