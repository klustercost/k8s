# klustercost Price Server

An internal microservice that resolves the hourly cost of Kubernetes nodes by querying cloud provider pricing APIs.

## Overview

The price server sits behind a `ClusterIP` service and exposes a simple HTTP interface. Other components (such as the `update` service) call it with a node's region, VM SKU, and OS to get back the spot retail price per hour.

Currently supported providers:

| Provider | Status          |
|----------|-----------------|
| Azure    | Supported       |
| AWS      | Not implemented |
| GCP      | Not implemented |

The active provider is selected via the `PROVIDER` environment variable.

## API

### `GET /get`

Returns spot pricing data for a given VM SKU.

**Query parameters:**

| Parameter | Description              | Example               |
|-----------|--------------------------|-----------------------|
| `region`  | Cloud region             | `eastus`              |
| `sku`     | VM instance type         | `Standard_D2ps_v6`   |
| `os`      | Operating system         | `linux` or `windows`  |

**Response:** JSON array where each entry contains:

```
[armSkuName, retailPrice, unitOfMeasure, armRegionName, meterName, productName]
```

**Example:**

```bash
curl "http://localhost:5001/get?region=eastus&sku=Standard_D2ps_v6&os=linux"
```

```json
[["Standard_D2ps_v6", 0.0192, "1 Hour", "eastus", "D2ps v6 Spot", "Virtual Machines Dps v6 Series"]]
```

### `GET /about`

Returns service identity info.

## How It Fits Together

1. Nodes are collected into the database with their Kubernetes labels but without pricing information.
2. The `update` service polls for nodes where `price_per_hour` is `NULL`.
3. It parses the node labels (`topology.kubernetes.io/region`, `node.kubernetes.io/instance-type`, `kubernetes.io/os`) and calls the price server's `/get` endpoint.
4. The price server queries the cloud provider's retail pricing API (e.g. [Azure Retail Prices](https://learn.microsoft.com/en-us/rest/api/cost-management/retail-prices/azure-retail-prices)) and returns spot pricing data.
5. The `update` service writes the resulting price back to the database.

## Manual Testing

Port-forward the service and query it directly:

```bash
kubectl port-forward svc/klustercost-price 5001:80 -n your-ns
curl "http://localhost:5001/get?region=eastus&sku=Standard_D2ps_v6&os=linux"
```

To find the correct values from your cluster's nodes:

```bash
kubectl get nodes -o custom-columns="\
NAME:.metadata.name,\
REGION:.metadata.labels.topology\.kubernetes\.io/region,\
SKU:.metadata.labels.node\.kubernetes\.io/instance-type,\
OS:.metadata.labels.kubernetes\.io/os"
```

## Configuration

| Environment Variable | Description                        | Example  |
|----------------------|------------------------------------|----------|
| `PROVIDER`           | Cloud provider to query prices for | `azure`  |

## Running Locally

```bash
pip install -r requirements.txt
PROVIDER=azure python main.py
```

The server starts on port `5001` using Waitress as the WSGI server.
