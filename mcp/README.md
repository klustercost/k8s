# MCP Server -- Natural Language Database Queries

This project runs a lightweight [MCP](https://modelcontextprotocol.io/) server that lets you ask plain English questions about a PostgreSQL database. Under the hood it uses OpenAI to translate your question into SQL, runs the query, and returns the results as JSON.

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
├── .dockerignore
├── Dockerfile.server       # Docker image for the MCP server
├── Dockerfile.client       # Docker image for the interactive client
├── requirements.txt        # Server Python dependencies
├── requirements-client.txt # Client Python dependencies
├── system_prompt.txt       # OpenAI system prompt (editable)
├── my_server.py            # The MCP server (runs the tools)
├── my_client.py            # Interactive terminal client
└── README.md               # You are here
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
| `PG_DATABASE`    | Name of the database to connect to               | `postgres`  |
| `PG_SCHEMA`      | Schema to read tables from (used for introspection) | `public`  |

## Running

You need **two terminals** (both with the virtual environment activated).

### Terminal 1 -- Start the server

```bash
python my_server.py
```

The server starts on `http://127.0.0.1:8000/mcp` and waits for connections.

### Terminal 2 -- Query the database

You have two options:

**Option A: Interactive terminal client**

```bash
python my_client.py
```

This opens an interactive prompt where you type questions in plain English and get results back immediately:

```
Connected to MCP server at http://localhost:8000/mcp
Type your question and press Enter. Type 'exit' to quit.

Question: Which pod consumed the most CPU in the last 1 hour?
[... JSON results ...]

Question: exit
Bye!
```

**Option B: MCP-compatible client**

Any MCP-compatible client (Cursor, Claude Desktop, etc.) can connect to `http://127.0.0.1:8000/mcp` and call the `ask_db` tool directly.

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

**The client** (`my_client.py`) is just a thin terminal wrapper. It reads your question from stdin, sends it over HTTP to the server, and prints the response. It has no knowledge of SQL, PostgreSQL, or OpenAI -- it's purely a pass-through.

**The server** (`my_server.py`) does all the work in four stages:

1. **Schema introspection** -- Queries `information_schema.columns` in PostgreSQL to get the current table names, column names, and data types. This happens on every request, so the server always reflects the latest database structure.
2. **SQL generation** -- Sends the schema + your question to OpenAI via the Chat Completions API. A system prompt (loaded from `system_prompt.txt`) tells the model the domain context, the table relationships, and the PostgreSQL syntax rules. OpenAI returns a raw `SELECT` query. It never sees your actual data -- only the table/column metadata.
3. **Query execution** -- Runs the generated SQL against PostgreSQL and packs the rows into dictionaries.
4. **Response** -- Returns the results as JSON back to the client.

```
 You type a question
  │
  ▼
 my_client.py ──HTTP──► my_server.py
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

From the `mcp/` directory:

```bash
# Server image
docker build -f Dockerfile.server -t your-registry/mcp-server:latest .

# Client image
docker build -f Dockerfile.client -t your-registry/mcp-client:latest .
```

### Pushing to a registry

```bash
docker push your-registry/mcp-server:latest
docker push your-registry/mcp-client:latest
```

Replace `your-registry` with your actual container registry (e.g. `ghcr.io/yourorg`, `youracr.azurecr.io`, etc.).

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
  your-registry/mcp-server:latest

# Client (interactive)
docker run -it --rm \
  -e MCP_SERVER_URL=http://mcp-server:8000/mcp \
  --link mcp-server \
  your-registry/mcp-client:latest \
  python my_client.py
```

## Kubernetes Deployment (Helm)

The MCP server and client are packaged as part of the `klustercost` Helm chart. Helm templates live in `helm/klustercost/templates/mcp/`.

### What gets deployed

| Resource | Name | Purpose |
| -------- | ---- | ------- |
| Deployment | `<release>-mcp-server` | Runs the MCP server, connects to PostgreSQL and OpenAI |
| Service | `<release>-mcp-server` | ClusterIP service on port 8000, used by the client |
| Deployment | `<release>-mcp-client` | Idle pod you exec into for interactive CLI queries |
| Secret | `<release>-mcp-secret` | Stores the OpenAI API key |

### Configuring values.yaml

Set your image registry and OpenAI key in the `mcp` section of `values.yaml`:

```yaml
mcp:
  enabled: true
  imagePullPolicy: Always

  server:
    image: your-registry/mcp-server:latest
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
    image: your-registry/mcp-client:latest
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
  --set mcp.server.image="your-registry/mcp-server:latest" \
  --set mcp.client.image="your-registry/mcp-client:latest"
```

### Querying the database via CLI

The client pod is an idle container you exec into. This is the primary way to interact with the MCP server from within the cluster:

```bash
# Find the client pod
kubectl get pods -l app=klustercost-mcp-client

# Exec into it and start the interactive client
kubectl exec -it deploy/klustercost-mcp-client -- python my_client.py
```

You'll see the interactive prompt:

```
Connected to MCP server at http://klustercost-mcp-server.<namespace>.svc.cluster.local:8000/mcp
Type your question and press Enter. Type 'exit' to quit.

Question: Which pod consumed the most CPU in the last 1 hour?
[... JSON results ...]
```

### Accessing the server from outside the cluster

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
- The system prompt lives in `system_prompt.txt` and is baked into the server image. Edit it and rebuild to change the AI's behavior.
- If the generated SQL fails, the error message will include the SQL that was attempted, so you can see what went wrong and rephrase your question.
