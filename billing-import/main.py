#!/usr/bin/env python3
"""CLI for billing-import (ingest + node allocation)."""

from __future__ import annotations

import argparse
import logging
import sys
from datetime import datetime

from allocate import run_allocation
from coverage import coverage_stats
from db import reset_billing_data, set_node_provider_id
from ingest import ingest_azure_csv


def _parse_dt(value: str) -> datetime:
    for fmt in ("%Y-%m-%d", "%Y-%m-%dT%H:%M:%S"):
        try:
            return datetime.strptime(value, fmt)
        except ValueError:
            continue
    raise argparse.ArgumentTypeError(f"invalid date: {value!r}")


def _print_coverage(stats: dict) -> None:
    print("Map coverage (fact_costs VM/VMSS → tbl_nodes.provider_id)")
    print(f"  vm_resources: {stats['vm_resources']}")
    print(f"  mapped_resources: {stats['mapped_resources']}")
    print(f"  unmapped_resources: {stats['unmapped_resources']}")
    print(f"  unmapped_cost: {stats['unmapped_cost']}")
    if stats["unmapped"]:
        print("\nUnmapped resource_ids:")
        for resource_id, line_count, cost in stats["unmapped"]:
            print(f"  {cost}\t{line_count} lines\t{resource_id}")


def cmd_ingest(args: argparse.Namespace) -> int:
    stats = ingest_azure_csv(args.file)
    print("Ingest OK")
    for key, value in stats.items():
        print(f"  {key}: {value}")
    if args.no_allocate:
        return 0
    _print_coverage(coverage_stats(provider=args.provider))
    return cmd_allocate(args)


def cmd_verify(args: argparse.Namespace) -> int:
    from db import connect

    conn = connect()
    cur = conn.cursor()
    print("Row counts:")
    for name, sql in [
        ("fact_costs", "SELECT COUNT(*) FROM billing.tbl_fact_costs"),
        (
            "tbl_nodes_with_provider_id",
            "SELECT COUNT(*) FROM klustercost.tbl_nodes WHERE provider_id IS NOT NULL",
        ),
        ("node_cost_allocation", "SELECT COUNT(*) FROM klustercost.tbl_node_cost_allocation"),
    ]:
        cur.execute(sql)
        print(f"  {name}: {cur.fetchone()[0]}")

    cur.execute(
        """
        SELECT resource_id, sku, effective_cost, pricing_model, reservation_id
        FROM billing.tbl_fact_costs
        ORDER BY id
        LIMIT 10
        """
    )
    print("\nSample fact_costs:")
    for row in cur.fetchall():
        print(f"  {row}")
    conn.close()
    return 0


def cmd_reset(args: argparse.Namespace) -> int:
    reset_billing_data()
    print("Reset OK (fact_costs + node_cost_allocation cleared)")
    return 0


def cmd_verify_map(args: argparse.Namespace) -> int:
    stats = coverage_stats(provider=args.provider)
    _print_coverage(stats)
    if args.strict and stats["unmapped_resources"] > 0:
        return 1
    return 0


def cmd_node_set_provider(args: argparse.Namespace) -> int:
    set_node_provider_id(node_name=args.node, provider_id=args.provider_id)
    print(f"Set provider_id on {args.node}")
    return 0


def cmd_allocate(args: argparse.Namespace) -> int:
    stats = run_allocation(
        provider=args.provider,
        window_start=args.start,
        window_end=args.end,
        replace=not args.no_replace,
    )
    print("Allocate OK")
    for key, value in stats.items():
        print(f"  {key}: {value}")
    return 0


def cmd_verify_allocation(args: argparse.Namespace) -> int:
    from db import connect

    conn = connect()
    cur = conn.cursor()
    print("Row counts:")
    for name, sql in [
        (
            "tbl_nodes_with_provider_id",
            "SELECT COUNT(*) FROM klustercost.tbl_nodes WHERE provider_id IS NOT NULL",
        ),
        ("node_cost_allocation", "SELECT COUNT(*) FROM klustercost.tbl_node_cost_allocation"),
    ]:
        cur.execute(sql)
        print(f"  {name}: {cur.fetchone()[0]}")

    cur.execute(
        """
        SELECT node_name, window_start::date,
               list_cost, effective_cost, amortized_cost, reservation_discount
        FROM klustercost.tbl_node_cost_allocation
        ORDER BY id
        LIMIT 10
        """
    )
    print("\nnode_cost_allocation (list / effective / amortized / savings):")
    for row in cur.fetchall():
        print(f"  {row}")

    cur.execute(
        """
        SELECT date_trunc('day', window_start) AS day,
               SUM(effective_cost) AS billed
        FROM klustercost.tbl_node_cost_allocation
        GROUP BY 1
        ORDER BY 1
        """
    )
    print("\ndaily node totals (effective):")
    for row in cur.fetchall():
        print(f"  {row}")
    conn.close()
    return 0


def main() -> int:
    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
    parser = argparse.ArgumentParser(description="KlusterCost billing-import")
    sub = parser.add_subparsers(dest="command", required=True)

    p_ingest = sub.add_parser(
        "ingest",
        help="Import Azure-style cost CSV and allocate to nodes (default)",
    )
    p_ingest.add_argument("file", help="Path to CSV export")
    p_ingest.add_argument(
        "--no-allocate",
        action="store_true",
        help="Ingest only; skip coverage report and allocate",
    )
    p_ingest.add_argument("--provider", default="azure")
    p_ingest.add_argument("--start", type=_parse_dt, default=None, help="YYYY-MM-DD")
    p_ingest.add_argument("--end", type=_parse_dt, default=None, help="exclusive end, YYYY-MM-DD")
    p_ingest.add_argument("--no-replace", action="store_true", help="Do not clear prior allocation rows")
    p_ingest.set_defaults(func=cmd_ingest)

    p_verify = sub.add_parser("verify", help="Row counts for core billing tables")
    p_verify.set_defaults(func=cmd_verify)

    p_reset = sub.add_parser("reset", help="Clear fact_costs + node_cost_allocation")
    p_reset.set_defaults(func=cmd_reset)

    p_vm = sub.add_parser("verify-map", help="VM/VMSS coverage via tbl_nodes.provider_id")
    p_vm.add_argument("--provider", default="azure")
    p_vm.add_argument("--strict", action="store_true", help="Exit 1 if any VM/VMSS unmapped")
    p_vm.set_defaults(func=cmd_verify_map)

    p_node = sub.add_parser("node", help="Node provider_id helpers (dev/kind)")
    node_sub = p_node.add_subparsers(dest="node_command", required=True)
    p_set = node_sub.add_parser("set-provider-id", help="Set provider_id on a node")
    p_set.add_argument("--node", required=True, help="Kubernetes node name")
    p_set.add_argument("--provider-id", required=True, help="Full Azure ResourceId")
    p_set.set_defaults(func=cmd_node_set_provider)

    p_alloc = sub.add_parser("allocate", help="Roll fact_costs up to node_cost_allocation")
    p_alloc.add_argument("--provider", default="azure")
    p_alloc.add_argument("--start", type=_parse_dt, default=None, help="YYYY-MM-DD")
    p_alloc.add_argument("--end", type=_parse_dt, default=None, help="exclusive end, YYYY-MM-DD")
    p_alloc.add_argument("--no-replace", action="store_true", help="Do not clear prior rows in window")
    p_alloc.set_defaults(func=cmd_allocate)

    p_va = sub.add_parser("verify-allocation", help="Node allocation row counts and samples")
    p_va.set_defaults(func=cmd_verify_allocation)

    args = parser.parse_args()
    try:
        return args.func(args)
    except Exception as exc:
        logging.error("%s", exc)
        return 1


if __name__ == "__main__":
    sys.exit(main())
