"""Parse Azure Cost Management style CSV exports into normalized dicts."""

from __future__ import annotations

import csv
import hashlib
import json
from datetime import datetime, timedelta
from typing import Any


PARSER_VERSION = "azure_csv_v2"


def _parse_date(value: str) -> datetime:
    value = (value or "").strip()
    for fmt in ("%Y-%m-%d", "%m/%d/%Y", "%Y-%m-%dT%H:%M:%S", "%Y-%m-%d %H:%M:%S"):
        try:
            return datetime.strptime(value, fmt)
        except ValueError:
            continue
    raise ValueError(f"unsupported date format: {value!r}")


def _float(value: str | None) -> float | None:
    if value is None or str(value).strip() == "":
        return None
    return float(str(value).replace(",", ""))


def _row_hash(row: dict[str, Any]) -> str:
    payload = json.dumps(row, sort_keys=True, default=str)
    return hashlib.sha256(payload.encode("utf-8")).hexdigest()


def _reservation_id_short(full_id: str | None) -> str | None:
    if not full_id or not str(full_id).strip():
        return None
    full_id = str(full_id).strip()
    if "/" in full_id:
        return full_id.rstrip("/").split("/")[-1]
    return full_id


def _resolve_costs(raw: dict[str, str], quantity: float | None) -> tuple[float | None, float | None, float | None]:
    """
    Map Azure export columns to list / effective / amortized.
    See: https://learn.microsoft.com/en-us/azure/cost-management-billing/automate/understand-usage-details-fields
    """
    effective = _float(
        raw.get("CostInBillingCurrency")
        or raw.get("Cost")
        or raw.get("PreTaxCost")
    )
    list_cost = _float(
        raw.get("paygCostInBillingCurrency")
        or raw.get("PaygCostInBillingCurrency")
        or raw.get("PayGPrice")
        or raw.get("payGPrice")
    )
    if list_cost is None and effective is not None:
        payg_price = _float(raw.get("PayGPrice") or raw.get("payGPrice"))
        if payg_price is not None and quantity:
            list_cost = payg_price * quantity
    if list_cost is None:
        list_cost = effective

    amortized = _float(
        raw.get("CostInAmortized")
        or raw.get("AmortizedCost")
        or raw.get("costInAmortized")
    )
    if amortized is None:
        amortized = effective

    return list_cost, effective, amortized


def parse_azure_csv(path: str) -> tuple[list[dict[str, Any]], dict[str, Any]]:
    """
    Returns (normalized_rows, metadata).
    Each normalized row is ready for tbl_fact_costs + optional reservation side effects.
    """
    normalized: list[dict[str, Any]] = []
    period_start: datetime | None = None
    period_end: datetime | None = None

    with open(path, newline="", encoding="utf-8-sig") as handle:
        reader = csv.DictReader(
            (line for line in handle if not line.lstrip().startswith("#")),
        )
        if not reader.fieldnames:
            raise ValueError("CSV has no header row")

        for line_number, raw in enumerate(reader, start=1):
            usage_date = _parse_date(raw.get("Date") or raw.get("UsageDate") or "")
            usage_start = usage_date
            usage_end = usage_date + timedelta(hours=1)

            if period_start is None or usage_start < period_start:
                period_start = usage_start
            if period_end is None or usage_end > period_end:
                period_end = usage_end

            quantity = _float(raw.get("Quantity"))
            list_cost, effective_cost, amortized_cost = _resolve_costs(raw, quantity)
            reservation_full = (raw.get("ReservationId") or "").strip() or None
            reservation_id = _reservation_id_short(reservation_full)
            pricing_model = (raw.get("PricingModel") or "").strip() or None
            charge_type = (raw.get("ChargeType") or "Usage").strip() or "Usage"

            tags_raw = raw.get("Tags") or ""
            tags: dict[str, Any] | None = None
            if tags_raw.strip():
                try:
                    tags = json.loads(tags_raw)
                except json.JSONDecodeError:
                    tags = {"raw": tags_raw}

            sku = (
                raw.get("Meter")
                or raw.get("MeterName")
                or raw.get("Product")
                or raw.get("MeterSubcategory")
                or ""
            ).strip() or None

            row = {
                "line_number": line_number,
                "content_hash": None,
                "raw_json": raw,
                "usage_start": usage_start,
                "usage_end": usage_end,
                "subscription_id": (raw.get("SubscriptionId") or "").strip() or None,
                "account_id": (raw.get("BillingAccountId") or raw.get("AccountId") or "").strip() or None,
                "region": (raw.get("ResourceLocation") or raw.get("Location") or "").strip() or None,
                "resource_id": (raw.get("ResourceId") or "").strip() or None,
                "resource_type": (raw.get("ResourceType") or "").strip() or None,
                "sku": sku,
                "usage_quantity": quantity,
                "usage_unit": (raw.get("UnitOfMeasure") or raw.get("Unit") or "").strip() or None,
                "list_cost": list_cost,
                "effective_cost": effective_cost,
                "amortized_cost": amortized_cost,
                "currency": (
                    raw.get("BillingCurrencyCode")
                    or raw.get("BillingCurrency")
                    or raw.get("Currency")
                    or "EUR"
                ).strip(),
                "pricing_model": pricing_model,
                "charge_type": charge_type,
                "reservation_id": reservation_id,
                "reservation_id_full": reservation_full,
                "tags": tags,
            }
            row["content_hash"] = _row_hash({"file": path, "line": line_number, "raw": raw})
            normalized.append(row)

    metadata = {
        "parser_version": PARSER_VERSION,
        "billing_period_start": period_start,
        "billing_period_end": period_end,
        "row_count": len(normalized),
    }
    return normalized, metadata
