"""Ingest pipeline: CSV → billing.tbl_fact_costs."""

from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

from db import insert_fact_cost, transaction
from parsers.azure_csv import PARSER_VERSION, parse_azure_csv

logger = logging.getLogger(__name__)


def ingest_azure_csv(path: str) -> dict[str, Any]:
    path = str(Path(path).resolve())
    if not Path(path).is_file():
        raise FileNotFoundError(path)

    rows, meta = parse_azure_csv(path)
    provider = "azure"

    stats = {
        "path": path,
        "parser_version": PARSER_VERSION,
        "billing_period_start": meta.get("billing_period_start"),
        "billing_period_end": meta.get("billing_period_end"),
        "rows_parsed": len(rows),
        "facts_inserted": 0,
        "facts_skipped": 0,
    }

    with transaction() as conn:
        cur = conn.cursor()
        for row in rows:
            if insert_fact_cost(cur, provider, row):
                stats["facts_inserted"] += 1
            else:
                stats["facts_skipped"] += 1

    logger.info("ingest complete: %s", stats)
    return stats
