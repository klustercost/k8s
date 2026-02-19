import os
import json
import psycopg2
from dotenv import load_dotenv
from openai import OpenAI
from fastmcp import FastMCP

load_dotenv()

# System prompt file
PROMPT_FILE = os.path.join(os.path.dirname(__file__), "system_prompt.txt")
with open(PROMPT_FILE, encoding="utf-8") as f:
    SYSTEM_PROMPT_TEMPLATE = f.read()

# --- Configuration from .env ---
OPENAI_API_KEY = os.getenv("OPENAI_API_KEY")
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")
PG_HOST = os.getenv("PG_HOST", "localhost")
PG_PORT = int(os.getenv("PG_PORT", "5432"))
PG_USER = os.getenv("PG_USER", "postgres")
PG_PASSWORD = os.getenv("PG_PASSWORD", "")
PG_DATABASE = os.getenv("PG_DATABASE", "klustercost")
PG_SCHEMA = os.getenv("PG_SCHEMA", "klustercost")

openai_client = OpenAI(api_key=OPENAI_API_KEY)
mcp = FastMCP("My MCP Server")


def get_pg_connection():
    return psycopg2.connect(
        host=PG_HOST,
        port=PG_PORT,
        user=PG_USER,
        password=PG_PASSWORD,
        dbname=PG_DATABASE,
    )


def get_schema_info() -> str:
    """Fetch table and column metadata from information_schema."""
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
        return "\n".join(lines)
    finally:
        if conn is not None:
            conn.close()


def generate_sql(question: str, schema: str) -> str:
    """Ask OpenAI to produce a read-only SQL query for the given question."""
    response = openai_client.chat.completions.create(
        model=OPENAI_MODEL,
        messages=[
            {"role": "system", "content": SYSTEM_PROMPT_TEMPLATE.format(schema=schema)},
            {"role": "user", "content": question},
        ],
        temperature=0,
    )
    content = response.choices[0].message.content
    if content is None:
        raise ValueError("OpenAI returned an empty response — no SQL was generated")
    return content.strip()


def run_query(sql: str) -> list[dict]:
    """Execute a SELECT query and return rows as list of dicts."""
    conn = None
    try:
        conn = get_pg_connection()
        with conn.cursor() as cur:
            cur.execute(sql)
            if cur.description is None:
                raise ValueError("Query returned no result set — only SELECT statements are supported")
            columns = [desc[0] for desc in cur.description]
            return [dict(zip(columns, row)) for row in cur.fetchall()]
    finally:
        if conn is not None:
            conn.close()


# --- MCP Tools ---

@mcp.tool
def ask_db(question: str) -> str:
    """Ask a natural-language question about the PostgreSQL database.

    The question is converted to SQL via OpenAI, executed, and the
    results are returned as JSON.
    """
    sql = None
    try:
        schema = get_schema_info()
        sql = generate_sql(question, schema)
        if sql.strip() == "REFUSE":
            return "Sorry, I can only answer questions about the Kubernetes cluster database."
        rows = run_query(sql)
    except Exception as e:
        sql_info = f"\nGenerated SQL was:\n{sql}" if sql else ""
        return f"Error: {e}{sql_info}"
    return json.dumps(rows, indent=2, default=str)


if __name__ == "__main__":
    mcp.run(
        transport="streamable-http",
        host=os.getenv("MCP_HOST", "0.0.0.0"),
        port=int(os.getenv("MCP_PORT", "8000")),
        path="/mcp",
    )
