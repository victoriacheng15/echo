# Echo

**Echo** is a dual-interface persistent memory layer designed to provide both AI agents and human operators with a contextual "brain." It functions as both a **Model Context Protocol (MCP)** server for autonomous agents and a **Command Line Interface (CLI)** for manual curation.

By bridging the gap between stateless AI reasoning and local persistence, Echo allows you to store and recall architectural preferences, project-specific snippets, and frequent instructions in a local SQLite database. Unlike standard LLM sessions that reset context every time the process exits, Echo creates a long-term memory (LTM) layer that ensures established preferences are automatically inherited in all future sessions, across any directory on your machine.

🌐 [Project Portal](https://victoriacheng15.github.io/echo/)

📚 [Documentation Hub: Architecture, ADRs & Operations](./docs/README.md)

---

## 📚 Project Evolution

This platform evolved through intentional phases. See the full journey with ADRs:

[View Complete Evolution Log](docs/README.md)

### Key Milestones

- **Ch 1: Persistence** – SQLite with WAL mode for high-concurrency tool calls.
- **Ch 2: Performance** – FTS5 Inverted Index for $O(\log n)$ keyword search (230x faster).
- **Ch 3: Workflow** – Custom Go-based static generator for living architectural documentation.
- **Ch 4: Analytics** – DuckDB integration for knowledge ROI and autonomous memory refinement.
- **Ch 5: Precision** – Surgical memory management with surrogate IDs to eliminate content collisions and ensure deterministic state control.
- **Ch 6: Dual-Interface** – Standard Flag CLI for human-in-the-loop curation, enabling manual storage, recall, and maintenance of the SQLite "Brain."

Each milestone links to Architecture Decision Records (ADRs) showing the *why* behind each change.

---

## 🛠️ Tech Stack & Architecture

The platform leverages a robust set of modern technologies for its core functions:

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/sqlite-%2307405e.svg?style=for-the-badge&logo=sqlite&logoColor=white)
![DuckDB](https://img.shields.io/badge/duckdb-%23FFF000.svg?style=for-the-badge&logo=duckdb&logoColor=black)
![Nix](https://img.shields.io/badge/NIX-5277C3.svg?style=for-the-badge&logo=NixOS&logoColor=white)

### System Architecture Overview

The diagram below illustrates the dual-interface architecture, where both AI agents and human operators interact with a unified knowledge state through shared business services.

```mermaid
graph TD
    subgraph "Interfaces"
        A[AI Agent] -- "JSON-RPC 2.0" --> B[MCP Transport Layer]
        H[Terminal] -- "Flag Commands" --> I[CLI Dispatcher]
    end

    subgraph "Core Logic"
        B -- "Strong Types" --> C[Shared Service Layer]
        I -- "Shared State" --> C
        C -- "Validation" --> D{Storage Engine}
    end

    subgraph "Persistence & Analytics"
        D -- "WAL Persistence" --> E[(SQLite)]
        D -- "Full-Text Search" --> F[(FTS5 Index)]
        D -- "Telemetry" --> G[(DuckDB Analytics)]
    end
```

---

## 🚀 Key Achievements & Capabilities

### 🧠 Persistent Software Architecture

- **Contextual Recall:** Automatically partitions memories into `project:<name>` and `global` scopes for precision retrieval based on the active workspace.
- **XDG Compliance:** Strictly adheres to Linux standards by storing the "brain" in `~/.local/share/echo/`, ensuring LTM survives binary updates.
- **Architectural Isolation:** Implemented "Thin Main" patterns to decouple MCP transport logic from the core business services.

### ⚡ Operational Performance

- **Sub-Millisecond Search:** Utilizes FTS5 virtual tables to ensure the AI's reasoning loop is never throttled by I/O during heavy context retrieval.
- **Atomic Integrity:** Guarantees data consistency via SQLite transactions and Write-Ahead Logging (WAL), even during catastrophic process exits.
- **Zero-Config Analytics:** Leverages DuckDB to generate usage insights without requiring external database infrastructure or complex setup.

### 🛡️ Engineering Standards

- **Contract Enforcement:** Validates memory payloads against strict JSON schemas and 8KB content limits to prevent context bloat.
- **Reproducible Runtimes:** Leverages **Nix** flakes to ensure the CGO-linked SQLite environment is identical across different local machines.
- **Decision Framework:** Adopted Architectural Decision Records (ADRs) to document system evolution and manage technical design debt.

---

## 🚀 Getting Started

<details>
<summary><b>Operational Guide</b></summary>

This guide will help you set up, configure, and verify the Echo MCP server.

### Prerequisites

Ensure you have the following installed on your system:

- [Go](https://go.dev/doc/install) (1.25+)
- [Nix](https://nixos.org/download.html) (optional, for reproducible toolchains)
- `make` (GNU Make)
- `sqlite3` CLI (for manual audits)

### 1. Build and Install

Echo requires CGO for SQLite support. Use the provided Makefile for a guaranteed build:

```bash
# Build the binary
make build

# Install to /usr/local/bin/echo-cli
make install
```

### 2. Configuration

Add Echo to your MCP client configuration (e.g., `~/.gemini/settings.json` or `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "echo": {
      "command": "/usr/local/bin/echo-cli",
      "args": ["--db", "/home/[user]/.local/share/echo/echo.db"]
    }
  }
}
```

### 3. CLI Operations (Manual Curation)

Echo now supports a dual-interface architecture. You can manually curate the "brain" from your terminal:

- **Recall Memories**
```bash
echo-cli recall -contexts "project:echo,global" -limit 5
```

- **Keyword Search**
```bash
echo-cli search -query "commit message standard"
```

- **Manual Storage**
```bash
echo-cli store -content "Always use tabs" -context "global" -type "directive" -tags "styling,go"
```

- **Database Maintenance**
```bash
echo-cli maintain --rebuild --sync
```

### 4. Verification & Observability

Once installed, you can verify performance and audit the "brain" directly:

- **Performance Audit**

```bash
make bench
```

- **Direct Database Access**

```bash
# View the last 5 things the AI learned
sqlite3 ~/.local/share/echo/echo.db "SELECT * FROM memories ORDER BY last_used DESC LIMIT 5;"
```

### 4. Testing

Maintain high operational standards by running the full test suite:

```bash
# Run all tests
make test

# Generate coverage report
make test-cov
```

</details>
>
