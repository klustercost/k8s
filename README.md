# KlusterCost

KlusterCost is an open-source Kubernetes cost visibility platform. It collects workload, node, service, and ownership metadata from a cluster, combines it with resource usage metrics and cloud pricing data, and stores the results in PostgreSQL for reporting, dashboards, chat interfaces, and BI workflows.

The project is designed to be deployed with Helm and to run inside an existing Kubernetes cluster.

## What It Provides

- **Cluster monitoring**: watches Kubernetes pods, nodes, services, and application controllers, then records resource and ownership data.
- **Usage metrics**: queries Prometheus for pod CPU and memory consumption.
- **Cloud pricing enrichment**: resolves node pricing through the pricing service. Azure pricing is currently supported; AWS and GCP are planned.
- **PostgreSQL persistence**: stores collected metrics, metadata, and pricing information in a relational schema.
- **User interfaces and integrations**: includes a web frontend, Microsoft Teams bot, MCP-based natural-language assistant, and Power BI-oriented integration support.
- **Helm packaging**: deploys the full stack with configurable images, storage, Prometheus endpoint, MCP, Teams, and Power BI settings.

## Architecture

At a high level, KlusterCost runs as a set of cooperating services:

1. The **monitor** service observes Kubernetes resources and queries Prometheus for usage metrics.
2. PostgreSQL stores cluster state, usage data, ownership details, and pricing fields.
3. The **price** service queries cloud provider retail pricing APIs.
4. The **update** service enriches node records with hourly pricing data.
5. Optional frontends expose the data through web, Teams, MCP, and Power BI workflows.

## Repository Layout

- `helm/klustercost`: Helm chart for deploying KlusterCost.
- `monitor`: Go-based Kubernetes observer and Prometheus integration.
- `price`: Python pricing service. Azure is implemented today.
- `update`: Python worker that updates stored node records with price data.
- `mcp`: MCP server and HTTP client for natural-language questions over the KlusterCost database.
- `wab`: web frontend service.
- `chat/teams`: Microsoft Teams bot integration.
- `patcher`: helper job used to update Power BI network allowlists.
- `bi`: Power BI supporting assets and hierarchy data.

## Prerequisites

Before installing KlusterCost, make sure the target environment has:

- A running Kubernetes cluster.
- Helm 3.x.
- Prometheus installed and reachable from inside the cluster.
- Kubernetes Metrics Server installed in the cluster.
- A StorageClass for PostgreSQL persistent storage, recommended for any non-demo installation.
- Access to the public GitHub Container Registry images, or a configured private registry mirror.

Optional features require additional configuration:

- MCP assistant: an OpenAI API key.
- Microsoft Teams bot: Azure tenant, app registration, and bot identity configuration.
- Power BI access: network rules that allow Power BI to reach the exposed database endpoint, if that workflow is enabled.

## Installation

Install the published Helm chart from GHCR:

```bash
helm install klustercost oci://ghcr.io/klustercost/k8s/klustercost --version 0.1.0
```

For production-like deployments, enable persistent PostgreSQL storage and set a StorageClass available in your cluster:

```bash
helm install klustercost oci://ghcr.io/klustercost/k8s/klustercost --version 0.1.0 \
  --set postgresql.pvc.enabled=true \
  --set postgresql.pvc.storageClass=managed-csi
```

If your Prometheus service uses a different name, namespace, or port, override the in-cluster endpoint:

```bash
helm install klustercost oci://ghcr.io/klustercost/k8s/klustercost --version 0.1.0 \
  --set prometheus.prometheusServerAddress=http://prometheus-server.prometheus.svc.cluster.local
```

You can also install from a local checkout:

```bash
git clone https://github.com/klustercost/k8s.git
cd k8s
helm install klustercost helm/klustercost
```

For the full values reference, see [`helm/README.md`](helm/README.md).

## Configuration Notes

- The bundled PostgreSQL deployment uses ephemeral storage by default. Enable `postgresql.pvc.enabled=true` before relying on collected data.
- The default Prometheus endpoint is `http://prometheus-server.prometheus.svc.cluster.local`. Adjust `prometheus.prometheusServerAddress` to match your cluster.
- The pricing service defaults to `price.provider=azure`.
- The MCP components are enabled by default, but require `mcp.openai.apiKey` to answer natural-language questions.
- Teams and Power BI integrations are enabled in the chart values, but require environment-specific identity and networking configuration.

## Development

Each component includes its own README or deployment artifacts where applicable:

- Monitor: Go 1.21 module under `monitor`.
- Python services: install dependencies from each component's `requirements.txt`.
- Docker images: component Dockerfiles are provided alongside the service source.
- Helm chart: templates and SQL initialization files are under `helm/klustercost`.

## Contributing

Contributions are welcome. Please open an issue or pull request with a clear description of the problem, proposed change, and any testing performed.

For code changes, keep pull requests focused and include updates to documentation or Helm values when behavior changes.

## License

This project is licensed under the terms of the repository license. See [`LICENSE`](LICENSE).
