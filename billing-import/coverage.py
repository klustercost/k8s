"""Report VM/VMSS invoice lines with no matching tbl_nodes.provider_id."""

from __future__ import annotations

from typing import Any

from db import transaction

VM_COSTS_FILTER = """
    fc.resource_id ILIKE '%%/virtualMachines/%%'
    OR fc.resource_id ILIKE '%%/virtualMachineScaleSets/%%'
    OR fc.resource_type ILIKE '%%virtualMachines%%'
    OR fc.resource_type ILIKE '%%virtualMachineScaleSets%%'
"""


def unmapped_vm_resources(*, provider: str = "azure") -> list[tuple]:
    with transaction() as conn:
        cur = conn.cursor()
        cur.execute(
            f"""
            SELECT fc.resource_id,
                   COUNT(*) AS line_count,
                   SUM(COALESCE(fc.effective_cost, 0)) AS unmapped_cost
            FROM billing.tbl_fact_costs fc
            LEFT JOIN klustercost.tbl_nodes n
              ON lower(n.provider_id) = lower(fc.resource_id)
            WHERE fc.provider = %s
              AND fc.resource_id IS NOT NULL
              AND n.idx IS NULL
              AND ({VM_COSTS_FILTER})
            GROUP BY fc.resource_id
            ORDER BY unmapped_cost DESC
            """,
            (provider,),
        )
        return cur.fetchall()


def coverage_stats(*, provider: str = "azure") -> dict[str, Any]:
    with transaction() as conn:
        cur = conn.cursor()
        cur.execute(
            f"""
            SELECT COUNT(DISTINCT fc.resource_id)
            FROM billing.tbl_fact_costs fc
            WHERE fc.provider = %s
              AND fc.resource_id IS NOT NULL
              AND ({VM_COSTS_FILTER})
            """,
            (provider,),
        )
        vm_resources = cur.fetchone()[0] or 0

        cur.execute(
            f"""
            SELECT COUNT(DISTINCT fc.resource_id)
            FROM billing.tbl_fact_costs fc
            INNER JOIN klustercost.tbl_nodes n
              ON lower(n.provider_id) = lower(fc.resource_id)
            WHERE fc.provider = %s
              AND fc.resource_id IS NOT NULL
              AND ({VM_COSTS_FILTER})
            """,
            (provider,),
        )
        mapped_resources = cur.fetchone()[0] or 0

    unmapped = unmapped_vm_resources(provider=provider)
    unmapped_cost = sum(row[2] or 0 for row in unmapped)
    return {
        "provider": provider,
        "vm_resources": vm_resources,
        "mapped_resources": mapped_resources,
        "unmapped_resources": vm_resources - mapped_resources,
        "unmapped_cost": unmapped_cost,
        "unmapped": unmapped,
    }
