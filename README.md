# Echo

**Echo** is a Model Context Protocol (MCP) server designed to provide AI agents with a persistent, contextual "brain." By bridging the gap between stateless AI reasoning and local filesystem persistence, Echo allows your AI agent to store and recall architectural preferences, project-specific snippets, and frequent instructions in a local SQLite database.

Unlike standard LLM sessions that reset context every time the process exits, Echo creates a long-term memory (LTM) layer. This ensures that established project preferences and instructions are automatically inherited by the AI in all future sessions, across any directory on your machine.

## 🚀 Key Features

- **Contextual Recall**: Automatically provides relevant memories based on your current project (`project:name`) or global preferences (`global`).
- **Persistent Logic**: Uses SQLite with **WAL (Write-Ahead Logging)** mode for high-concurrency and reliability during simultaneous tool calls.
- **XDG Compliant**: Automatically stores your "brain" in `~/.local/share/echo/echo.db` ensuring memories survive project deletions or binary updates.
- **Reproducible Environment**: Fully integrated with **Makefile** and **Nix** for consistent builds and CGO-linked SQLite execution.
- **High Integrity**: Enforces strict data contracts, including 8KB content limits, metadata JSON validation, and enumerated entry types.

## 🏗️ Tech Stack

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/sqlite-%2307405e.svg?style=for-the-badge&logo=sqlite&logoColor=white)
![Nix](https://img.shields.io/badge/NIX-5277C3.svg?style=for-the-badge&logo=NixOS&logoColor=white)

## 🛠️ Quick Start

### 1. Build & Install

Echo requires Go 1.25+ and CGO (for SQLite). The provided Makefile automatically detects and uses Nix if available, ensuring a guaranteed build environment.

```bash
# Build the binary
make build

# Install to ~/.local/bin/echo
make install
```

### 2. Configuration

Add Echo to your Gemini CLI configuration (usually located at `~/.gemini/settings.json`):

```json
{
  "mcpServers": {
    "echo": {
      "command": "/home/[user]/.local/bin/echo",
      "args": ["--db", "/home/[user]/.local/share/echo/echo.db"]
    }
  }
}
```

## 🧠 MCP Tools Interface

Echo provides four core tools to your AI agent:

1. **`store_memory`**: Saves or updates a contextual memory. Performs an `UPSERT` if the content already exists for a given key.
2. **`recall_memory`**: Retrieves the most relevant memories for the current environment based on usage count and recency.
3. **`search_memories`**: Full-text search across the entire persistent "brain."
4. **`delete_memory`**: Manually prunes a specific memory from the database.

## 🧪 Testing & Quality

Echo maintains high operational standards with exhaustive error-path testing and **>80% statement coverage**:

```bash
# Run all tests (automatically uses Nix if available)
make test

# Run tests with coverage report
make test-cov
```

## 🔍 Observability (CLI "God Mode")

Since Echo uses standard SQLite, you can audit your AI's memories directly from your terminal using the `sqlite3` CLI.

If you don't have it installed:

- **Ubuntu/Debian**: `sudo apt install sqlite3`
- **macOS**: `brew install sqlite`
- **Nix**: `nix-shell -p sqlite`

```bash
# View the last 5 things the AI learned
sqlite3 ~/.local/share/echo/echo.db "SELECT * FROM memories ORDER BY last_used DESC LIMIT 5;"
```
