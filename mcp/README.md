# MCP Server -- Natural Language Database Queries

This project runs a lightweight [MCP](https://modelcontextprotocol.io/) server that lets you ask plain English questions about a PostgreSQL database. Under the hood it uses OpenAI to translate your question into SQL, runs the query, and returns the results as JSON.

It also exposes a JSONata-to-PostgreSQL DDL generator. That endpoint accepts a JSONata object expression and returns SQL text for a table, composite type, and indexes. It does not execute the generated DDL.

Built with [FastMCP](https://github.com/jlowin/fastmcp), a Python framework that makes it easy to create MCP-compatible servers with minimal boilerplate.

## Prerequisites

- **Python 3.10+**
- **A PostgreSQL database** you can connect to (local or remote)
- **An OpenAI API key** -- get one at https://platform.openai.com/api-keys

## Project Structure

```
mcp/
├── .env                    # Your credentials (never committed to git)
├── .gitignore
├── README.md               # You are here
├── server/
│   ├── Dockerfile          # Docker image for the MCP server
│   ├── requirements.txt    # Server Python dependencies
│   ├── my_server.py        # The MCP server (runs the tools)
│   ├── jsonata_ddl_prompt.txt # JSONata-to-DDL prompt template
│   └── system_prompt.txt   # OpenAI system prompt (editable)
└── client/
    ├── Dockerfile          # Docker image for the HTTP client
    ├── requirements.txt    # Client Python dependencies
    └── my_client.py        # HTTP server that exposes the /ask endpoint
```

## Setup

### 1. Create and activate a virtual environment

```bash
# From the mcp/ directory
python -m venv .venv

# Windows
.venv\Scripts\activate

# macOS / Linux
source .venv/bin/activate
```

### 2. Install dependencies

```bash
pip install -r requirements.txt
```

### 3. Configure your credentials

Open the `.env` file and replace the placeholder values with your real credentials:

```
# OpenAI
OPENAI_API_KEY=sk-proj-...your-real-key...

# PostgreSQL
PG_HOST=your-host
PG_PORT=5432
PG_USER=postgres
PG_PASSWORD=your-password
PG_DATABASE=klustercost
PG_SCHEMA=klustercost
```

| Variable         | Description                                      | Default     |
| ---------------- | ------------------------------------------------ | ----------- |
| `OPENAI_API_KEY` | Your OpenAI API key (required)                   | --          |
| `PG_HOST`        | PostgreSQL server hostname                       | `localhost` |
| `PG_PORT`        | PostgreSQL server port                           | `5432`      |
| `PG_USER`        | Database user                                    | `postgres`  |
| `PG_PASSWORD`    | Database password                                | (empty)     |
| `PG_DATABASE`    | Name of the database to connect to               | `klustercost` |
| `PG_SCHEMA`      | Schema to read tables from and default DDL schema | `klustercost` |

## Running

You need **two terminals** (both with the virtual environment activated).

### Terminal 1 -- Start the server

```bash
python my_server.py
```

The server starts on `http://127.0.0.1:8000/mcp` and waits for connections.

### Terminal 2 -- Query the database

You have two options:

**Option A: HTTP client endpoint**

```bash
python my_client.py
```

This starts an HTTP server on `http://0.0.0.0:8080` that accepts questions and JSONata DDL requests via REST endpoints:

```bash
curl -X POST http://localhost:8080/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Which pod consumed the most CPU in the last 1 hour?"}'
```

Generate PostgreSQL DDL from JSONata:

```bash
curl -X POST http://localhost:8080/translate-jsonata \
  -H "Content-Type: application/json" \
  -d '{
    "jsonata": "{ \"uid\":metadata.uid, \"name\":metadata.name, \"namespace\":metadata.namespace, \"node\": spec.nodeName }",
    "table_name": "tbl_pods",
    "schema": "klustercost",
    "index_columns": ["uid", "namespace", "node"]
  }'
```

See the [HTTP Endpoint](#http-endpoint) section below for full details.

**Option B: MCP-compatible client**

Any MCP-compatible client (Cursor, Claude Desktop, etc.) can connect to `http://127.0.0.1:8000/mcp` and call the `ask_db` or `translate_jsonata_to_psql` tools directly.

## Example Questions

You write plain English -- the system figures out the SQL for you. Here are some examples to get you started:

```
"Which pod consumed the most CPU in the last 1 hour?"
"Show me the average memory usage per namespace"
"What are the top 5 pods by CPU usage today?"
"List all pods in the default namespace"
"How many data points were recorded in the last 24 hours?"
```

You do **not** need to know the exact table or column names. The server reads the database schema automatically and sends it to OpenAI so it can generate the correct query.

## How It Works

The system has two parts: a **client** and a **server**.

**The client** (`my_client.py`) is a lightweight FastAPI HTTP server. It exposes `POST /ask` and `POST /translate-jsonata`, forwards requests to the MCP server over the MCP protocol, and returns the result as JSON. It has no knowledge of SQL, PostgreSQL, or OpenAI -- it's a pass-through bridge.

**The server** (`my_server.py`) does all the work in four stages:

1. **Schema introspection** -- Queries `information_schema.columns` in PostgreSQL to get the current table names, column names, and data types. This happens on every request, so the server always reflects the latest database structure.
2. **SQL generation** -- Sends the schema + your question to OpenAI via the Responses API. A system prompt (loaded from `system_prompt.txt`) tells the model the domain context, the table relationships, and the PostgreSQL syntax rules. OpenAI returns a raw `SELECT` query. It never sees your actual data -- only the table/column metadata.
3. **Query execution** -- Runs the generated SQL against PostgreSQL and packs the rows into dictionaries.
4. **Response** -- Returns the results as JSON back to the client.

```
 curl POST /ask
  │
  ▼
 my_client.py ──MCP protocol──► my_server.py
  (HTTP :8080)                    (HTTP :8000/mcp)
                                       │
                              ┌────────┼────────┐
                              ▼                  ▼
                         PostgreSQL          OpenAI
                       (read schema)    (generate SQL)
                              │                  │
                              └───────┬──────────┘
                                      ▼
                            Execute generated SQL
                                      │
                                      ▼
                            JSON results back to client
```

## HTTP Endpoint

The MCP client (`my_client.py`) runs an HTTP server that acts as a bridge between plain HTTP requests and the MCP server. It accepts a natural-language question, forwards it to the MCP server via the MCP protocol, and returns the AI-generated result as JSON.

### Available Endpoints

| Method | Path       | Description                              |
|--------|------------|------------------------------------------|
| POST   | `/ask`     | Send a natural-language question to the AI |
| POST   | `/translate-jsonata` | Generate PostgreSQL DDL from a JSONata object expression |
| GET    | `/healthz` | Health check (returns `{"status": "ok"}`) |

### Sending a Question

```bash
curl -X POST http://localhost:8080/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "What are the top 5 pods by CPU usage?"}'
```

Successful response (HTTP 200):

```json
{
  "answer": "[{\"name\": \"pod-abc\", \"cpu\": 0.85}, ...]"
}
```

### Generating DDL from JSONata

Use `POST /translate-jsonata` when you have a JSONata object expression and want PostgreSQL DDL for storing the transformed objects.

The endpoint calls the MCP server tool `translate_jsonata_to_psql`. The MCP server calls OpenAI, parses strict JSON from the model, validates the generated SQL, and returns the DDL text. The DDL is never executed.

Step by step:

1. Start the MCP server from `mcp/server`:

```bash
python my_server.py
```

2. Start the HTTP client from `mcp/client`:

```bash
python my_client.py
```

3. Send a request to the client:

```bash
curl -X POST http://localhost:8080/translate-jsonata \
  -H "Content-Type: application/json" \
  -d '{
    "jsonata": "{ \"uid\":metadata.uid, \"name\":metadata.name, \"namespace\":metadata.namespace, \"node\": spec.nodeName, \"app.name\":metadata.labels.`app.kubernetes.io/name`, \"app.component\":metadata.labels.`app.kubernetes.io/component` }",
    "table_name": "tbl_pods",
    "schema": "klustercost",
    "index_columns": ["uid", "namespace", "node", "app.name", "app.component"]
  }'
```

PowerShell example using the repo's pod labels JSONata file:

```powershell
$jsonata = [string](Get-Content .\helm\klustercost\transform\pod\labels.jsonata -Raw)

$body = @{
  jsonata = $jsonata
  table_name = "tbl_pods"
  schema = "klustercost"
  index_columns = @("uid", "namespace", "node", "app.name", "app.component")
} | ConvertTo-Json

Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/translate-jsonata `
  -ContentType "application/json" `
  -Body $body
```

The `[string](...)` cast is important in Windows PowerShell. Without it, `ConvertTo-Json` can serialize `Get-Content` as an object with metadata instead of a plain JSONata string.

Request fields:

| Field | Required | Description |
| ----- | -------- | ----------- |
| `jsonata` | Yes | JSONata object expression. The object keys become SQL columns. |
| `table_name` | Yes | Target table name. Must be a plain PostgreSQL identifier, for example `tbl_pods`. |
| `type_name` | No | Composite type name. If omitted, `tbl_pods` becomes `pod_type`. |
| `schema` | No | Target schema. Defaults to `PG_SCHEMA`, which defaults to `klustercost`. |
| `index_columns` | No | Column names that should have indexes when possible. Dotted names such as `app.name` are allowed as column names. |

Example successful response:

```json
{
  "status": "success",
  "ddl": "CREATE TABLE IF NOT EXISTS klustercost.tbl_pods (...);\nCREATE TYPE klustercost.pod_type AS (...);\nCREATE INDEX IF NOT EXISTS tbl_pods_uid ON klustercost.tbl_pods USING hash (uid);",
  "table_name": "tbl_pods",
  "type_name": "pod_type",
  "schema": "klustercost",
  "warnings": []
}
```

The generated SQL should include:

- `CREATE TABLE IF NOT EXISTS klustercost.tbl_pods`
- `CREATE TYPE klustercost.pod_type AS`
- `CREATE INDEX IF NOT EXISTS ... ON klustercost.tbl_pods`
- Quoted dotted column names, for example `"app.name"`

Validation rules:

- The model must return strict JSON with `ddl` and `warnings`.
- `CREATE SCHEMA` is not allowed.
- Only `CREATE TABLE IF NOT EXISTS`, `CREATE TYPE`, and `CREATE INDEX IF NOT EXISTS` are allowed.
- `DROP`, `DELETE`, `UPDATE`, `INSERT`, `ALTER`, `TRUNCATE`, `CALL`, `COPY`, `DO`, `GRANT`, `REVOKE`, and `EXECUTE` are rejected.
- The generated DDL must target the requested schema, table name, and type name.
- References to other schemas are rejected.
- SQL comments are rejected.

If validation fails, the endpoint returns a JSON body with `status: "error"` and an `error` message. It still does not execute anything.

### Calling the MCP Tool Directly

MCP-compatible clients can call `translate_jsonata_to_psql` directly on `http://localhost:8000/mcp`.

Tool arguments:

```json
{
  "jsonata": "{ \"uid\":metadata.uid, \"name\":metadata.name }",
  "table_name": "tbl_pods",
  "type_name": null,
  "schema": "klustercost",
  "index_columns": ["uid"]
}
```

The tool returns a JSON string using the same response shape as `POST /translate-jsonata`.

### Error Responses

| Status | Meaning                                    | Example Body                                        |
|--------|--------------------------------------------|-----------------------------------------------------|
| 400    | Bad request (missing or invalid JSON body) | `{"error": "Missing or empty 'question' field"}`    |
| 404    | Unknown endpoint                           | `{"error": "Not found"}`                            |
| 500    | Server-side error                          | `{"error": "Internal server error"}`                |

### Configuration

| Variable          | Default                      | Description                       |
|-------------------|------------------------------|-----------------------------------|
| `MCP_SERVER_URL`  | `http://localhost:8000/mcp`  | URL of the MCP server             |
| `MCP_CLIENT_HOST` | `0.0.0.0`                   | Address the HTTP server binds to  |
| `MCP_CLIENT_PORT` | `8080`                       | Port the HTTP server listens on   |
| `LOG_LEVEL`       | `INFO`                       | Log level (DEBUG or INFO)         |

## Troubleshooting

| Problem | Fix |
| ------- | --- |
| `ModuleNotFoundError: No module named 'fastmcp'` | Make sure you activated the virtual environment before running |
| `connection refused` from the MCP client | Make sure the server is running first |
| `FATAL: password authentication failed` | Check `PG_USER` and `PG_PASSWORD` in `.env` |
| `FATAL: database "..." does not exist` | Check `PG_DATABASE` in `.env` |
| OpenAI `AuthenticationError` | Check that `OPENAI_API_KEY` in `.env` is valid |
| Results are empty or wrong | Try rephrasing your question, or mention specific table/column names if you know them |

## Docker Images

Two separate images are provided -- one for the server and one for the client. Both use `python:3.14-slim-bookworm` as a base and run as a non-root user for security.

### Building the images

From the repository root:

```bash
# Server image
docker build -t ghcr.io/klustercost/k8s/klustercost-mcp-server:latest mcp/server/

# Client image
docker build -t ghcr.io/klustercost/k8s/klustercost-mcp-client:latest mcp/client/
```

### Pushing to a registry

```bash
docker push ghcr.io/klustercost/k8s/klustercost-mcp-server:latest
docker push ghcr.io/klustercost/k8s/klustercost-mcp-client:latest
```

### Running with Docker locally

```bash
# Server
docker run -d --name mcp-server \
  -e OPENAI_API_KEY=sk-... \
  -e PG_HOST=host.docker.internal \
  -e PG_USER=klustercost \
  -e PG_PASSWORD=klustercost \
  -e PG_DATABASE=klustercost \
  -e PG_SCHEMA=klustercost \
  -p 8000:8000 \
  ghcr.io/klustercost/k8s/klustercost-mcp-server:latest

# Client (HTTP server on port 8080)
docker run -d --name mcp-client \
  -e MCP_SERVER_URL=http://mcp-server:8000/mcp \
  --link mcp-server \
  -p 8080:8080 \
  ghcr.io/klustercost/k8s/klustercost-mcp-client:latest
```

## Kubernetes Deployment (Helm)

The MCP server and client are packaged as part of the `klustercost` Helm chart. Helm templates live in `helm/klustercost/templates/mcp/`.

### What gets deployed

| Resource | Name | Purpose |
| -------- | ---- | ------- |
| Deployment | `<release>-mcp-server` | Runs the MCP server, connects to PostgreSQL and OpenAI |
| Service | `<release>-mcp-server` | ClusterIP service on port 8000, used by the client |
| Deployment | `<release>-mcp-client` | Runs the HTTP server that exposes the /ask endpoint |
| Service | `<release>-mcp-client` | LoadBalancer service on port 8080, exposed externally |
| Secret | `<release>-mcp-secret` | Stores the OpenAI API key |

### Configuring values.yaml

Set your image registry and OpenAI key in the `mcp` section of `values.yaml`:

```yaml
mcp:
  enabled: true
  imagePullPolicy: Always

  server:
    image: ghcr.io/klustercost/k8s/klustercost-mcp-server:latest
    replicas: 1
    port: 8000
    resources:
      requests:
        cpu: 50m
        memory: 128Mi
      limits:
        cpu: 500m
        memory: 512Mi

  client:
    image: ghcr.io/klustercost/k8s/klustercost-mcp-client:latest
    port: 8080
    serviceType: LoadBalancer
    resources:
      requests:
        cpu: 50m
        memory: 64Mi
      limits:
        cpu: 200m
        memory: 256Mi

  openai:
    apiKey: "sk-proj-your-key-here"
    model: "gpt-4o-mini"

  postgresql:
    schema: "klustercost"
```

PostgreSQL connection details (host, port, user, password, database) are automatically inherited from the existing `postgresql` section in values.yaml. The server connects to the in-cluster PostgreSQL service.

### Deploying

```bash
helm upgrade --install klustercost helm/klustercost/ \
  --set mcp.openai.apiKey="sk-proj-your-key-here" \
  --set mcp.server.image="ghcr.io/klustercost/k8s/klustercost-mcp-server:latest" \
  --set mcp.client.image="ghcr.io/klustercost/k8s/klustercost-mcp-client:latest"
```

### Querying the database via the HTTP endpoint

The client pod exposes a `POST /ask` endpoint via a LoadBalancer service. Get the external IP and send requests:

```bash
# Get the external IP of the client service
kubectl get svc -l app=klustercost-mcp-client

# Send a question
curl -X POST http://<EXTERNAL-IP>:8080/ask \
  -H "Content-Type: application/json" \
  -d '{"question": "Which pod consumed the most CPU in the last 1 hour?"}'
```

See the [HTTP Endpoint](#http-endpoint) section for full endpoint documentation.

### Accessing the MCP server directly from outside the cluster

If you want to connect to the MCP server from your local machine (e.g. with Cursor or Claude Desktop):

```bash
kubectl port-forward svc/klustercost-mcp-server 8000:8000
```

Then point your MCP client to `http://localhost:8000/mcp`.

### Disabling MCP

Set `mcp.enabled: false` in values.yaml (or `--set mcp.enabled=false`) to skip deploying the MCP components entirely.

## Notes

- Only **read-only** (`SELECT`) queries are generated and executed. The system will not modify your data.
- The OpenAI model is configurable via `mcp.openai.model` in values.yaml or `OPENAI_MODEL` env var (default: `gpt-4o-mini`).
- The database question prompt lives in `system_prompt.txt`; the JSONata DDL prompt lives in `jsonata_ddl_prompt.txt`. Both are baked into the server image. Edit them and rebuild to change the AI's behavior.
- If the generated SQL fails, the error message will include the SQL that was attempted, so you can see what went wrong and rephrase your question.
