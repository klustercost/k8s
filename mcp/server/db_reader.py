import json
import logging
import os
import time
from functools import lru_cache

import psycopg2
from dotenv import load_dotenv
from openai import OpenAI

from answer_formatter import format_answer

load_dotenv(override=False)

log = logging.getLogger("mcp-server")

PROMPT_FILE = os.path.join(os.path.dirname(__file__), "system_prompt.txt")
with open(PROMPT_FILE, encoding="utf-8") as f:
    SYSTEM_PROMPT_TEMPLATE = f.read()

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

REFUSE_MESSAGE = (
    "Sorry, I can only answer questions about the Kubernetes cluster database. "
    "Please try again with a question about the cluster. "
    "For example: Which pods are using the most CPU today?"
)


def get_pg_connection():
    return psycopg2.connect(
        host=PG_HOST,
        port=PG_PORT,
        user=PG_USER,
        password=PG_PASSWORD,
        dbname=PG_DATABASE,
    )


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


def generate_sql(question: str, schema: str, response_id: str = None) -> tuple[str, str]:
    """Ask OpenAI to produce a read-only SQL query for the given question."""
    log.info("Generating SQL via OpenAI (model=%s) …", OPENAI_MODEL)
    t0 = time.perf_counter()

    response = openai_client.responses.create(
        model=OPENAI_MODEL,
        previous_response_id=response_id,
        instructions=SYSTEM_PROMPT_TEMPLATE.format(schema=schema),
        input=question,
    )

    elapsed = time.perf_counter() - t0
    content = response.output_text
    if content is None:
        raise ValueError("OpenAI returned an empty response — no SQL was generated")
    sql = content.strip()
    log.info(f"SQL generated in {elapsed:.2f}:\n{sql}")
    return sql, response.id


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


def process_question(question: str, response_id: str | None) -> dict:
    """Run the full ask_db flow: schema lookup, SQL generation, query, and formatting."""
    sql = None
    try:
        schema = get_schema_info()
        sql, response_id = generate_sql(question, schema, response_id)
        if sql.strip() == "REFUSE":
            log.warning("Question refused by LLM (off-topic)")
            return {
                "raw": REFUSE_MESSAGE,
                "natural": REFUSE_MESSAGE,
                "status": "refused",
            }
        rows = run_query(sql)
    except Exception as e:
        sql_info = f"\nGenerated SQL was:\n{sql}" if sql else ""
        log.error("ask_db failed: %s%s", e, sql_info)
        error_text = f"Error: {e}{sql_info}"
        return {
            "raw": error_text,
            "natural": error_text,
            "status": "error",
        }

    natural = None
    try:
        natural = format_answer(
            question,
            json.dumps(rows, default=str),
            openai_client,
            OPENAI_ANSWER_MODEL,
        )
    except Exception as e:
        log.error("Answer formatting failed: %s", e)

    return {
        "response_id": response_id,
        "raw": rows,
        "natural": natural,
        "status": "success",
    }
