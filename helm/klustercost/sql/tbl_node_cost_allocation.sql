CREATE SCHEMA IF NOT EXISTS klustercost;

CREATE TABLE IF NOT EXISTS klustercost.tbl_node_cost_allocation (
    id                      BIGSERIAL PRIMARY KEY,
    node_name               TEXT NOT NULL,
    provider_resource_id    TEXT,
    window_start            TIMESTAMP NOT NULL,
    window_end              TIMESTAMP NOT NULL,
    list_cost               NUMERIC,
    effective_cost          NUMERIC,
    amortized_cost          NUMERIC,
    reservation_discount    NUMERIC DEFAULT 0,
    currency                TEXT DEFAULT 'USD',
    pricing_source          TEXT NOT NULL DEFAULT 'invoice'
        CHECK (pricing_source IN ('invoice', 'list_price', 'effective_rate_blend')),
    provider                TEXT CHECK (provider IN ('azure', 'aws', 'gcp')),
    created_at              TIMESTAMP NOT NULL DEFAULT now(),
    CONSTRAINT tbl_node_cost_allocation_window CHECK (window_end > window_start)
);

CREATE INDEX IF NOT EXISTS idx_tbl_node_cost_allocation_node_window
    ON klustercost.tbl_node_cost_allocation (node_name, window_start, window_end);

CREATE INDEX IF NOT EXISTS idx_tbl_node_cost_allocation_provider_resource
    ON klustercost.tbl_node_cost_allocation (provider_resource_id)
    WHERE provider_resource_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_tbl_node_cost_allocation_dedupe
    ON klustercost.tbl_node_cost_allocation (node_name, provider_resource_id, window_start, window_end)
    WHERE provider_resource_id IS NOT NULL;
