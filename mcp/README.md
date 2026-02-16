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
├── .env                # Your credentials (never committed to git)
├── .gitignore
├── requirements.txt    # Python dependencies
├── my_server.py        # The MCP server (runs the tools)
├── my_client.py        # Interactive terminal client
└── README.md           # You are here
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

1. **You** send a plain English question to the `ask_db` tool via any MCP-compatible client.
2. **The server** connects to PostgreSQL and reads the schema -- all table names, column names, and data types for the configured schema.
3. **The server** sends the schema and your question to OpenAI (`model-of-your-choice`), which generates a `SELECT` SQL query.
4. **The server** executes the SQL against PostgreSQL.
5. **You** receive the query results as JSON.

```
 You (question)
  │
  ▼
 MCP Client (Cursor, Claude Desktop, etc.)
  │
  ──── HTTP ────►  MCP Server
                       │
             ┌─────────┼─────────┐
             ▼                    ▼
        PostgreSQL            OpenAI
      (read schema)     (generate SQL)
             │                    │
             └────────┬───────────┘
                      ▼
              Execute SQL query
                      │
                      ▼
             Return JSON results
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

## Notes

- Only **read-only** (`SELECT`) queries are generated and executed. The system will not modify your data.
- The OpenAI model used is `gpt-4o-mini`. You can change this in `my_server.py` in the `generate_sql()` function.
- If the generated SQL fails, the error message will include the SQL that was attempted, so you can see what went wrong and rephrase your question.
