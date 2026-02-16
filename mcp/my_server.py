import os
import json
import psycopg2
from dotenv import load_dotenv
from openai import OpenAI
from fastmcp import FastMCP

load_dotenv()

# --- Configuration from .env ---
OPENAI_API_KEY = os.getenv("OPENAI_API_KEY")
PG_HOST = os.getenv("PG_HOST", "localhost")
PG_PORT = os.getenv("PG_PORT", "5432")
PG_USER = os.getenv("PG_USER", "postgres")
PG_PASSWORD = os.getenv("PG_PASSWORD", "")
PG_DATABASE = os.getenv("PG_DATABASE", "postgres")
PG_SCHEMA = os.getenv("PG_SCHEMA", "public")

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
    conn = get_pg_connection()
    try:
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
    finally:
        conn.close()

    tables: dict[str, list[str]] = {}
    for table, column, dtype in rows:
        tables.setdefault(table, []).append(f"{column} ({dtype})")

    lines = []
    for table, cols in tables.items():
        lines.append(f"{PG_SCHEMA}.{table}: {', '.join(cols)}")
    return "\n".join(lines)


def generate_sql(question: str, schema: str) -> str:
    """Ask OpenAI to produce a read-only SQL query for the given question."""
    response = openai_client.chat.completions.create(
        model="gpt-4o-mini",
        messages=[
            {
                "role": "system",
                "content": (
                    "You are a SQL assistant. Given the database schema below, "
                    "write a single PostgreSQL SELECT query that answers the "
                    "user's question. Return ONLY the raw SQL — no markdown, "
                    "no explanation, no code fences.\n\n"
                    f"Schema:\n{schema}"
                ),
            },
            {"role": "user", "content": question},
        ],
        temperature=0,
    )
    return response.choices[0].message.content.strip()


def run_query(sql: str) -> list[dict]:
    """Execute a SELECT query and return rows as list of dicts."""
    conn = get_pg_connection()
    try:
        with conn.cursor() as cur:
            cur.execute(sql)
            columns = [desc[0] for desc in cur.description]
            return [dict(zip(columns, row)) for row in cur.fetchall()]
    finally:
        conn.close()


# --- MCP Tools ---

@mcp.tool
def greet(name: str) -> str:
    """Greet someone by name."""
    return f"Hello, {name}!"


@mcp.tool
def ask_db(question: str) -> str:
    """Ask a natural-language question about the PostgreSQL database.

    The question is converted to SQL via OpenAI, executed, and the
    results are returned as JSON.
    """
    schema = get_schema_info()
    sql = generate_sql(question, schema)
    try:
        rows = run_query(sql)
    except Exception as e:
        return f"SQL error: {e}\nGenerated SQL was:\n{sql}"
    return json.dumps(rows, indent=2, default=str)


if __name__ == "__main__":
    mcp.run(transport="streamable-http", host="127.0.0.1", port=8000, path="/mcp")
