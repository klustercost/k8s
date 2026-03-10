# klustercost Update Service

A background worker that enriches Kubernetes node records with hourly pricing data by coordinating between the database and the price server.

## Overview

The update service runs as a long-lived loop inside the cluster. Every 10 seconds it queries PostgreSQL for nodes that have labels but no price yet, resolves the cost through the [price server](../price/README.md), and writes the result back.

It acts as the glue between raw node metadata (collected by the monitor) and the pricing information served by the price server.

```
┌──────────┐       ┌──────────────┐       ┌──────────────┐
│ PostgreSQL│◄─────►│ update       │──────►│ price server │
│ tbl_nodes │       │ (this svc)   │       │ /get         │
└──────────┘       └──────────────┘       └──────────────┘
```

## How It Works

1. **Connect** -- On startup the service connects to PostgreSQL using environment variables and resolves the price server URI.
2. **Poll** -- Every 10 seconds it queries `klustercost.tbl_nodes` for rows where `price_per_hour IS NULL` and `labels IS NOT NULL`.
3. **Parse labels** -- Each node's label string is split into key-value pairs. Three labels are extracted:

   | Label | Purpose |
   |-------|---------|
   | `topology.kubernetes.io/region` | Cloud region (e.g. `eastus`) |
   | `node.kubernetes.io/instance-type` | VM SKU (e.g. `Standard_D2ps_v6`) |
   | `kubernetes.io/os` | Operating system (`linux` / `windows`) |

4. **Fetch price** -- The extracted values are sent to the price server's `/get` endpoint. The response contains the spot retail price per hour.
5. **Update** -- The price is written back to `price_per_hour` for that node row.
6. **Cache** -- Results are cached in memory by the full label string so identical nodes are only looked up once.

## Configuration

| Environment Variable | Description | Example |
|----------------------|-------------|---------|
| `price_uri` | Hostname (or host:port) of the price server | `klustercost-price` |
| `host` | PostgreSQL host | `klustercost-postgres-service` |
| `database` | PostgreSQL database name | `klustercost` |
| `user` | PostgreSQL user | `postgres` |
| `password` | PostgreSQL password | `postgres` |
| `port` | PostgreSQL port | `5432` |

## Running Locally

Make sure PostgreSQL and the price server are reachable, then:

```bash
pip install -r requirements.txt
```

Set the environment variables (or source the dev config):

```bash
export $(grep -v '^#' config/env | xargs)
python main.py
```

The service will start polling immediately and log each price lookup.

## Deployment

The service is deployed as a single-replica Kubernetes `Deployment` with no exposed ports -- it only makes outbound connections to the database and the price server.

**Standalone manifest:** `yaml/deployment.yaml`

**Helm:** `helm/klustercost/templates/klustercost/update-deployment.yaml` -- the Helm chart wires the price server and database URIs using the release name and namespace automatically.

## TODO

1. Set currency -- prices are returned in USD by default.
2. Map environment variables to a Kubernetes Secret.
3. Set currency it is in USD by default