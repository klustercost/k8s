# billing-import

CLI to ingest **Azure Cost Management–style CSV** into Postgres, then allocate VM/VMSS invoice lines to Kubernetes nodes.

**Manual:** export/upload CSV and run `ingest`. **Automatic after ingest:** coverage report + allocation (unless `--no-allocate`).

Mapping uses `klustercost.tbl_nodes.provider_id` (from monitor `Node.spec.providerID` on AKS; set manually on kind).

## Tables

| Table | Role |
|-------|------|
| `billing.tbl_fact_costs` | Normalized invoice lines |
| `klustercost.tbl_nodes` | Nodes; `provider_id` = Azure `ResourceId` |
| `klustercost.tbl_node_cost_allocation` | Billed cost per node per calendar day |

Schema: `helm/klustercost/sql/tbl_fact_costs.sql`, `tbl_node_cost_allocation.sql`, `tbl_nodes.sql`. Apply if tables are missing.

`pip install -r requirements.txt`. Postgres via env or `.env` (`PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`, `PGDATABASE`).

## Workflow

### AKS / prod

Monitor writes `provider_id` on nodes. Export CSV from Azure, then:

```bash
python main.py ingest /path/to/azure_cost_export.csv
python main.py verify-allocation
```

### kind / dev

Nodes often lack a real Azure `ResourceId`. Set `provider_id` to match a `ResourceId` in your CSV before ingest:

```bash
python main.py reset
python main.py node set-provider-id \
  --node 'klustercost-control-plane' \
  --provider-id '/subscriptions/.../virtualMachines/aks-nodepool-abc'
python main.py ingest /path/to/azure_cost_export.csv
python main.py verify-allocation
```

**Re-ingest:** fact rows dedupe on `(provider, content_hash)`; allocate refreshes rows in the billing window.

## CLI

| Command | Purpose |
|---------|---------|
| `ingest <file>` | Parse Azure CSV → `tbl_fact_costs`; then coverage + allocate |
| `ingest <file> --no-allocate` | Ingest only |
| `allocate` | Roll facts → `tbl_node_cost_allocation` (`--start`, `--end`, `--provider azure`) |
| `verify-map` | VM/VMSS lines with no matching `provider_id` (`--strict` exits 1 if unmapped) |
| `verify` | Row counts + sample `tbl_fact_costs` |
| `verify-allocation` | Row counts + sample allocation rows |
| `reset` | `TRUNCATE` fact_costs + node_cost_allocation |
| `node set-provider-id` | Dev helper: set `tbl_nodes.provider_id` |

Ingest is **Azure CSV only** today (`provider` is always `azure` on insert).

## Azure CSV export

Portal → **Cost Management + Billing** → **Cost analysis** → export CSV.

Expected columns (with fallbacks in the parser): `Date`, `ResourceId`, `ResourceLocation`, `CostInBillingCurrency` (or payg/amortized variants), `Quantity`, `Meter`, `PricingModel`, optional `ReservationId`, `Tags`.

Default currency when missing in CSV: **EUR**.
