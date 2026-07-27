"""Phase 2: tbl_fact_costs + tbl_nodes.provider_id → tbl_node_cost_allocation."""

from __future__ import annotations

import logging
from datetime import datetime
from typing import Any

from coverage import VM_COSTS_FILTER
from db import transaction

logger = logging.getLogger(__name__)


def _default_window(cur) -> tuple[datetime, datetime]:
    cur.execute(
        """
        SELECT date_trunc('day', MIN(usage_start)),
               date_trunc('day', MAX(usage_start)) + interval '1 day'
        FROM billing.tbl_fact_costs
        """
    )
    row = cur.fetchone()
    if not row or row[0] is None:
        raise ValueError("no tbl_fact_costs rows; run ingest first")
    return row[0], row[1]


def _clear_allocations(cur, window_start: datetime, window_end: datetime) -> None:
    cur.execute(
        """
        DELETE FROM klustercost.tbl_node_cost_allocation
        WHERE window_start < %s AND window_end > %s
        """,
        (window_end, window_start),
    )


def allocate_node_costs(
    cur,
    *,
    provider: str,
    window_start: datetime,
    window_end: datetime,
) -> int:
    cur.execute(
        f"""
        INSERT INTO klustercost.tbl_node_cost_allocation (
            node_name, provider_resource_id, window_start, window_end,
            list_cost, effective_cost, amortized_cost, reservation_discount,
            currency, pricing_source, provider
        )
        SELECT
            n.node,
            fc.resource_id,
            date_trunc('day', fc.usage_start),
            date_trunc('day', fc.usage_start) + interval '1 day',
            SUM(COALESCE(fc.list_cost, 0)),
            SUM(COALESCE(fc.effective_cost, 0)),
            SUM(COALESCE(fc.amortized_cost, fc.effective_cost, 0)),
            SUM(GREATEST(COALESCE(fc.list_cost, 0) - COALESCE(fc.effective_cost, 0), 0)),
            MAX(fc.currency),
            'invoice',
            fc.provider
        FROM billing.tbl_fact_costs fc
        INNER JOIN klustercost.tbl_nodes n
            ON lower(n.provider_id) = lower(fc.resource_id)
        WHERE fc.provider = %s
          AND fc.resource_id IS NOT NULL
          AND n.provider_id IS NOT NULL
          AND fc.usage_start >= %s
          AND fc.usage_start < %s
          AND ({VM_COSTS_FILTER})
        GROUP BY
            n.node,
            fc.resource_id,
            fc.provider,
            date_trunc('day', fc.usage_start)
        """,
        (provider, window_start, window_end),
    )
    return cur.rowcount


def run_allocation(
    *,
    provider: str = "azure",
    window_start: datetime | None = None,
    window_end: datetime | None = None,
    replace: bool = True,
) -> dict[str, Any]:
    stats: dict[str, Any] = {
        "provider": provider,
        "window_start": None,
        "window_end": None,
        "node_rows": 0,
    }

    with transaction() as conn:
        cur = conn.cursor()
        if window_start is None or window_end is None:
            window_start, window_end = _default_window(cur)
        stats["window_start"] = window_start
        stats["window_end"] = window_end

        if replace:
            _clear_allocations(cur, window_start, window_end)

        stats["node_rows"] = allocate_node_costs(
            cur,
            provider=provider,
            window_start=window_start,
            window_end=window_end,
        )

    logger.info("allocation complete: %s", stats)
    return stats
