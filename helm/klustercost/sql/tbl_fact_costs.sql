CREATE SCHEMA IF NOT EXISTS billing;

CREATE TABLE IF NOT EXISTS billing.tbl_fact_costs (
    id                  BIGSERIAL PRIMARY KEY,
    provider            TEXT NOT NULL CHECK (provider IN ('azure', 'aws', 'gcp')),
    usage_start         TIMESTAMP NOT NULL,
    usage_end           TIMESTAMP NOT NULL,
    subscription_id     TEXT,
    account_id          TEXT,
    region              TEXT,
    resource_id         TEXT,
    resource_type       TEXT,
    sku                 TEXT,
    usage_quantity      NUMERIC,
    usage_unit          TEXT,
    list_cost           NUMERIC,
    effective_cost      NUMERIC,
    amortized_cost      NUMERIC,
    currency            TEXT DEFAULT 'USD',
    pricing_model       TEXT,
    charge_type         TEXT,
    reservation_id      TEXT,
    tags                JSONB,
    content_hash        TEXT,
    imported_at         TIMESTAMP NOT NULL DEFAULT now(),
    CONSTRAINT tbl_fact_costs_window CHECK (usage_end >= usage_start)
);

CREATE INDEX IF NOT EXISTS idx_tbl_fact_costs_usage_window
    ON billing.tbl_fact_costs (usage_start, usage_end);

CREATE INDEX IF NOT EXISTS idx_tbl_fact_costs_resource
    ON billing.tbl_fact_costs (provider, resource_id);

CREATE INDEX IF NOT EXISTS idx_tbl_fact_costs_region_sku
    ON billing.tbl_fact_costs (provider, region, sku);

CREATE INDEX IF NOT EXISTS idx_tbl_fact_costs_reservation
    ON billing.tbl_fact_costs (reservation_id)
    WHERE reservation_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_tbl_fact_costs_account
    ON billing.tbl_fact_costs (subscription_id, account_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tbl_fact_costs_content_hash
    ON billing.tbl_fact_costs (provider, content_hash)
    WHERE content_hash IS NOT NULL;
