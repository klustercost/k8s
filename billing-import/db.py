"""PostgreSQL helpers for billing ingest."""

from __future__ import annotations

import os
from contextlib import contextmanager
from typing import Any, Iterator

import psycopg2
import psycopg2.extras
from dotenv import load_dotenv

load_dotenv(override=False)


def connect():
    return psycopg2.connect(
        host=os.getenv("PGHOST", os.getenv("host", "127.0.0.1")),
        port=os.getenv("PGPORT", os.getenv("port", "5432")),
        user=os.getenv("PGUSER", os.getenv("user", "klustercost")),
        password=os.getenv("PGPASSWORD", os.getenv("password", "klustercost")),
        database=os.getenv("PGDATABASE", os.getenv("database", "klustercost")),
    )


@contextmanager
def transaction() -> Iterator[Any]:
    conn = connect()
    try:
        yield conn
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()


def reset_billing_data() -> None:
    """Clear billing ingest + node allocation tables (not telemetry)."""
    with transaction() as conn:
        cur = conn.cursor()
        cur.execute("TRUNCATE klustercost.tbl_node_cost_allocation RESTART IDENTITY")
        cur.execute("TRUNCATE billing.tbl_fact_costs RESTART IDENTITY")


def set_node_provider_id(*, node_name: str, provider_id: str) -> None:
    """Set provider_id on a node (dev/kind override)."""
    with transaction() as conn:
        cur = conn.cursor()
        cur.execute(
            """
            UPDATE klustercost.tbl_nodes
            SET provider_id = %s
            WHERE node = %s
            """,
            (provider_id.strip(), node_name),
        )
        if cur.rowcount == 0:
            raise ValueError(f"node not found: {node_name!r}")


def insert_fact_cost(cur, provider: str, row: dict[str, Any]) -> bool:
    """Insert one fact row. Returns True if inserted, False if duplicate."""
    cur.execute(
        """
        INSERT INTO billing.tbl_fact_costs (
            provider, usage_start, usage_end, subscription_id, account_id,
            region, resource_id, resource_type, sku, usage_quantity, usage_unit,
            list_cost, effective_cost, amortized_cost, currency, pricing_model,
            charge_type, reservation_id, tags, content_hash
        ) VALUES (
            %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
        )
        ON CONFLICT (provider, content_hash) WHERE content_hash IS NOT NULL DO NOTHING
        RETURNING id
        """,
        (
            provider,
            row["usage_start"],
            row["usage_end"],
            row["subscription_id"],
            row["account_id"],
            row["region"],
            row["resource_id"],
            row["resource_type"],
            row["sku"],
            row["usage_quantity"],
            row["usage_unit"],
            row["list_cost"],
            row["effective_cost"],
            row["amortized_cost"],
            row["currency"],
            row["pricing_model"],
            row["charge_type"],
            row["reservation_id"],
            psycopg2.extras.Json(row["tags"]) if row["tags"] is not None else None,
            row["content_hash"],
        ),
    )
    return cur.fetchone() is not None
