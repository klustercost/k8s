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
PG_PORT = int(os.getenv("PG_PORT", "5432"))
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
    finally:
        if conn is not None:
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
                    "You are a SQL assistant for a Kubernetes cluster monitoring system.\n\n"
                    "Domain context:\n"
                    "- tbl_pods contains metadata about pods running in the cluster "
                    "(name, namespace, node, app labels, etc.).\n"
                    "- tbl_pod_data contains time-series metrics collected every 10 minutes. "
                    "Each row has a timestamp plus cpu and memory usage for one pod.\n"
                    "- tbl_pod_data.idx_pod is a foreign key referencing tbl_pods.idx.\n"
                    "- To get a pod's name alongside its metrics, JOIN tbl_pod_data ON "
                    "tbl_pod_data.idx_pod = tbl_pods.idx.\n\n"
                    "SQL rules:\n"
                    "- Always double-quote column names that contain dots or hyphens "
                    '(e.g. "app.name", "app.part-of").\n'
                    "- Always qualify table names with the schema (e.g. klustercost.tbl_pods).\n"
                    "- Write a single PostgreSQL SELECT query. No INSERT/UPDATE/DELETE.\n"
                    "- Return ONLY the raw SQL — no markdown, no explanation, no code fences.\n\n"
                    f"Schema:\n{schema}"
                ),
            },
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
        rows = run_query(sql)
    except Exception as e:
        sql_info = f"\nGenerated SQL was:\n{sql}" if sql else ""
        return f"Error: {e}{sql_info}"
    return json.dumps(rows, indent=2, default=str)


if __name__ == "__main__":
    mcp.run(transport="streamable-http", host="127.0.0.1", port=8000, path="/mcp")
