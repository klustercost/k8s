import json
import logging
import os
import re
import time

from dotenv import load_dotenv
from openai import OpenAI

load_dotenv(override=False)

log = logging.getLogger("mcp-server")

JSONATA_DDL_PROMPT_FILE = os.path.join(os.path.dirname(__file__), "jsonata_ddl_prompt.txt")
with open(JSONATA_DDL_PROMPT_FILE, encoding="utf-8") as f:
    JSONATA_DDL_PROMPT_TEMPLATE = f.read()

OPENAI_API_KEY = os.getenv("OPENAI_API_KEY")
OPENAI_MODEL = os.getenv("OPENAI_MODEL", "gpt-5.4-mini")
PG_SCHEMA = os.getenv("PG_SCHEMA", "klustercost")

openai_client = OpenAI(api_key=OPENAI_API_KEY)

IDENTIFIER_RE = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")
DISALLOWED_DDL_RE = re.compile(
    r"\b(DROP|DELETE|UPDATE|INSERT|ALTER|TRUNCATE|CALL|COPY|DO|GRANT|REVOKE|EXECUTE|CREATE\s+SCHEMA)\b",
    re.IGNORECASE,
)


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


def process_jsonata_ddl_request(
    jsonata: str,
    table_name: str,
    type_name: str | None = None,
    schema: str | None = None,
    index_columns: list[str] | None = None,
) -> dict:
    """Validate inputs and generate PostgreSQL DDL from a JSONata object expression."""
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
        if not isinstance(index_columns, list) or not all(
            isinstance(item, str) and item.strip() for item in index_columns
        ):
            raise ValueError("'index_columns' must be a list of non-empty strings")

    ddl, warnings = generate_jsonata_ddl(
        jsonata.strip(),
        table_name,
        resolved_type_name,
        resolved_schema,
        index_columns,
    )
    return {
        "status": "success",
        "ddl": ddl,
        "table_name": table_name,
        "type_name": resolved_type_name,
        "schema": resolved_schema,
        "warnings": warnings,
    }
