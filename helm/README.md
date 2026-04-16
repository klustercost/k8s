# KlusterCost Helm Chart

Deploys KlusterCost into an existing Kubernetes cluster. KlusterCost monitors resource usage and cost across your workloads and exposes the data through a web frontend, Teams bot, MCP-based AI assistant, and Power BI integration.

## Install from OCI registry

```bash
helm install klustercost oci://ghcr.io/klustercost/k8s/klustercost --version 0.1.0
```

To pull without installing:

```bash
helm pull oci://ghcr.io/klustercost/k8s/klustercost --version 0.1.0
```

## Override values

```bash
helm install klustercost oci://ghcr.io/klustercost/k8s/klustercost --version 0.1.0 \
  --set postgresql.pvc.enabled=true \
  --set postgresql.pvc.storageClass=managed-csi \
  -f my-values.yaml
```

---

## Values Reference

### `global`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `global.appName` | string | `"klustercost"` | Application name used in `app.kubernetes.io/name` labels across all resources. |
| `global.extraLabels` | object | `{}` | Arbitrary key/value pairs added to every resource's labels. Useful for organisation-wide tagging (e.g. `team`, `cost-center`). |

### `imageRegistry` — Private Container Registry

Set these when your cluster pulls images from a private registry instead of the public GHCR.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `imageRegistry.enabled` | bool | `false` | When `true`, creates an `imagePullSecret` and attaches it to every pod so images can be pulled from a private registry. |
| `imageRegistry.server` | string | `""` | Hostname of the private registry (e.g. `myregistry.azurecr.io`). |
| `imageRegistry.username` | string | `""` | Username for registry authentication. |
| `imageRegistry.password` | string | `""` | Password (or token) for registry authentication. |

### `postgresql` — PostgreSQL Database

KlusterCost stores all collected metrics in a PostgreSQL instance deployed as a StatefulSet inside the cluster.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `postgresql.enabled` | bool | `true` | Deploy the bundled PostgreSQL StatefulSet. Set to `false` only if you supply your own external database. |
| `postgresql.image` | string | `"postgres:17"` | Docker image (and tag) for PostgreSQL. |
| `postgresql.name` | string | `"klustercost"` | Name of the default database created on first startup. |
| `postgresql.username` | string | `"klustercost"` | Database user created on first startup. |
| `postgresql.password` | string | `"klustercost"` | Password for the database user. **Change this in production.** |
| `postgresql.port` | int | `5432` | Port exposed by the PostgreSQL service. |
| `postgresql.serviceType` | string | `"ClusterIP"` | Kubernetes Service type for PostgreSQL (`ClusterIP`, `NodePort`, `LoadBalancer`). |

#### `postgresql.tls` — Database TLS

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `postgresql.tls.enabled` | bool | `false` | When `true`, PostgreSQL starts with SSL enabled and mounts a TLS secret (`postgres-tls`) containing the certificate and key. |

#### `postgresql.pvc` — Persistent Storage

> **Important:** When `postgresql.pvc.enabled` is `false` (the default), PostgreSQL uses an `emptyDir` volume — a **transparent mode** where **all data is lost** whenever the pod restarts or gets rescheduled. Always enable a PVC for production workloads.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `postgresql.pvc.enabled` | bool | `false` | Create a PersistentVolumeClaim for PostgreSQL data. **Set to `true` for production.** |
| `postgresql.pvc.size` | string | `"8Gi"` | Requested storage size for the PVC. |
| `postgresql.pvc.storageClass` | string | `"managed-csi"` | StorageClass to use for the PVC. Must be a StorageClass available in your cluster (e.g. `managed-csi` on AKS, `gp3` on EKS, `standard` on GKE). |

### `monitor` — Cluster Monitor

The monitor component watches Kubernetes resources and records usage metrics into PostgreSQL.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `monitor.image` | string | `"ghcr.io/klustercost/k8s/klustercost-monitor:latest"` | Docker image for the monitor deployment. |
| `monitor.resyncTime` | int | `300` | Interval in **seconds** between full resync cycles of cluster state. Lower values increase data freshness but add API server load. |
| `monitor.workers` | int | `3` | Number of concurrent worker goroutines that process resource events. |

### `price` — Pricing Engine

Fetches cloud provider pricing data so KlusterCost can compute actual costs.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `price.image` | string | `"ghcr.io/klustercost/k8s/klustercost-price:latest"` | Docker image for the pricing deployment. |
| `price.provider` | string | `"azure"` | Cloud provider to pull pricing from. Supported values: `azure`, `aws`, `gcp`. |

### `update` — Update Service

Handles schema migrations and data update jobs.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `update.image` | string | `"ghcr.io/klustercost/k8s/klustercost-update:latest"` | Docker image for the update deployment. |

### `prometheus` — Prometheus Integration

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `prometheus.prometheusServerAddress` | string | `"http://prometheus-server.prometheus.svc.cluster.local"` | In-cluster URL of the Prometheus server used to query resource utilisation metrics. Adjust if your Prometheus is in a different namespace or uses a different service name. |

### `mcp` — MCP AI Assistant

The MCP (Model Context Protocol) components provide an AI-powered assistant that can answer natural-language questions about your cluster costs. It consists of a **server** (talks to PostgreSQL and the LLM) and a **client** (exposes a chat UI).

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `mcp.enabled` | bool | `true` | Deploy the MCP server and client. |
| `mcp.imagePullPolicy` | string | `"Always"` | Kubernetes `imagePullPolicy` for MCP pods. |
| `mcp.logLevel` | string | `"INFO"` | Log verbosity for MCP components. Valid values: `DEBUG`, `INFO`. |

#### `mcp.server`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `mcp.server.image` | string | `"ghcr.io/klustercost/k8s/klustercost-mcp-server:latest"` | Docker image for the MCP server. |
| `mcp.server.replicas` | int | `1` | Number of MCP server replicas. |
| `mcp.server.port` | int | `8000` | Port the MCP server listens on. |
| `mcp.server.resources.requests.cpu` | string | `"50m"` | CPU request for the MCP server pod. |
| `mcp.server.resources.requests.memory` | string | `"128Mi"` | Memory request for the MCP server pod. |
| `mcp.server.resources.limits.cpu` | string | `"500m"` | CPU limit for the MCP server pod. |
| `mcp.server.resources.limits.memory` | string | `"512Mi"` | Memory limit for the MCP server pod. |

#### `mcp.client`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `mcp.client.image` | string | `"ghcr.io/klustercost/k8s/klustercost-mcp-client:latest"` | Docker image for the MCP client. |
| `mcp.client.port` | int | `8080` | Port the MCP client UI listens on. |
| `mcp.client.serviceType` | string | `"ClusterIP"` | Kubernetes Service type for the MCP client. |
| `mcp.client.sessionTimeout` | int | `3600` | Chat session timeout in **seconds**. After this period of inactivity the session is dropped. |
| `mcp.client.resources.requests.cpu` | string | `"50m"` | CPU request for the MCP client pod. |
| `mcp.client.resources.requests.memory` | string | `"64Mi"` | Memory request for the MCP client pod. |
| `mcp.client.resources.limits.cpu` | string | `"200m"` | CPU limit for the MCP client pod. |
| `mcp.client.resources.limits.memory` | string | `"256Mi"` | Memory limit for the MCP client pod. |

#### `mcp.openai` — OpenAI Configuration

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `mcp.openai.apiKey` | string | `""` | OpenAI API key. **Required** for the MCP AI features to work. |
| `mcp.openai.model` | string | `"gpt-5.4-mini"` | OpenAI model used for tool-calling and reasoning inside the MCP server. |
| `mcp.openai.answerModel` | string | `"gpt-5.4-mini"` | OpenAI model used for formatting the final user-facing answer. Can differ from `model` to trade speed for quality. |

#### `mcp.postgresql`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `mcp.postgresql.schema` | string | `"klustercost"` | PostgreSQL schema the MCP server reads cost data from. |

### `wafrontend` — Web Frontend

A web-based dashboard for browsing cost data.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `wafrontend.enabled` | bool | `true` | Deploy the web frontend. |
| `wafrontend.image` | string | `"ghcr.io/klustercost/k8s/klustercost-wab:latest"` | Docker image for the frontend. |
| `wafrontend.port` | int | `80` | Service port exposed externally. |
| `wafrontend.targetPort` | int | `5000` | Port the container listens on internally. |
| `wafrontend.serviceType` | string | `"ClusterIP"` | Kubernetes Service type (`ClusterIP`, `NodePort`, `LoadBalancer`). |
| `wafrontend.replicas` | int | `1` | Number of frontend replicas. |
| `wafrontend.resources.requests.cpu` | string | `"50m"` | CPU request. |
| `wafrontend.resources.requests.memory` | string | `"64Mi"` | Memory request. |
| `wafrontend.resources.limits.cpu` | string | `"200m"` | CPU limit. |
| `wafrontend.resources.limits.memory` | string | `"256Mi"` | Memory limit. |
| `wafrontend.accesstoken` | string | `""` | Access token for webhook verification (platform-specific). |
| `wafrontend.verifytoken` | string | `""` | Verify token for webhook verification (platform-specific). |
| `wafrontend.template` | string | `"standard_simple_reply"` | Response template name controlling the frontend's reply format. |
| `wafrontend.templateContext` | string | `""` | Additional context injected into the response template. |

### `powerbi` — Power BI IP Allowlist

A CronJob that keeps a NetworkPolicy or firewall rule up to date with Microsoft Power BI's published IP ranges, so Power BI can reach the database.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `powerbi.enabled` | bool | `true` | Deploy the Power BI IP updater CronJob. |
| `powerbi.image` | string | `"ghcr.io/klustercost/k8s/klustercost-patcher:latest"` | Docker image for the patcher job. |
| `powerbi.configUri` | string | *(Azure Service Tags URL)* | URL to the Microsoft Service Tags JSON file. Updated periodically by Microsoft. |
| `powerbi.extract` | string | `"$.values[?(@.name == 'PowerBI')].properties.addressPrefixes.*"` | JSONPath expression used to extract Power BI IP CIDRs from the Service Tags file. |
| `powerbi.additionalIps` | string | `"{}"` | JSON array of extra CIDRs to allowlist alongside the automatically fetched Power BI ranges (e.g. `'["1.2.3.4/32"]'`). Use `"{}"` for none. |

### `teams` — Microsoft Teams Bot

A bot that lets users query cost data directly from Microsoft Teams.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `teams.enabled` | bool | `true` | Deploy the Teams bot. |
| `teams.image` | string | `"ghcr.io/klustercost/k8s/klustercost-teamsb:latest"` | Docker image for the Teams bot. |
| `teams.replicas` | int | `1` | Number of bot replicas. |
| `teams.port` | int | `80` | Service port exposed externally. |
| `teams.targetPort` | int | `8080` | Port the container listens on internally. |
| `teams.serviceType` | string | `"ClusterIP"` | Kubernetes Service type. |
| `teams.resources.requests.cpu` | string | `"50m"` | CPU request. |
| `teams.resources.requests.memory` | string | `"64Mi"` | Memory request. |
| `teams.resources.limits.cpu` | string | `"200m"` | CPU limit. |
| `teams.resources.limits.memory` | string | `"256Mi"` | Memory limit. |
| `teams.tenantId` | string | `""` | Azure AD tenant ID for the bot registration. **Required** for the bot to authenticate. |
| `teams.clientId` | string | `""` | Azure AD app (client) ID for the bot registration. **Required** for the bot to authenticate. |
| `teams.botType` | string | `"UserAssignedMsi"` | Bot authentication type. `UserAssignedMsi` uses a user-assigned managed identity; change if your setup uses a different credential type. |
