# System Architecture

The Echo MCP Server is designed with a strict separation of concerns, ensuring that the transport protocol (MCP) is entirely decoupled from the business logic and persistence layers. This allows for high testability and future expansion (e.g., adding a CLI interface).

## High-Level Data Flow

1. **Client (AI Agent / IDE):** Sends a JSON-RPC 2.0 request over STDIO.
2. **Transport Layer (`cmd/mcp`):** The `mark3labs/mcp-go` server parses the request, validates the tool name, and extracts the arguments.
3. **Business Logic (`internal/service`):** The `MemoryService` validates the payload (e.g., enforcing the 8KB limit, checking context key regex) and determines the appropriate database interaction.
4. **Persistence Layer (`internal/db`):** Executes the SQL query against the local `echo.db` SQLite file.

## Core Components

### 1. Transport Layer (`cmd/mcp`)

- Handles the STDIO lifecycle.
- Registers the 5 core tools with the MCP framework:
  - `store_memory`
  - `recall_memory`
  - `search_memories`
  - `update_memory`
  - `delete_memory`
- Maps raw JSON arguments to strong Go types before passing them to the service.

### 2. Business Logic (`internal/service`)

- **`MemoryService`**: The core application logic. It holds no knowledge of the MCP protocol.
- **Hybrid Search**: Implements the logic to route short queries (< 3 chars) to a `LIKE` scan and longer queries to the FTS5 index.
- **Validation**: Enforces strict data contracts (`entry_type` enums, valid JSON metadata) to ensure the AI does not corrupt the "brain".

### 3. Persistence Layer (`internal/db`)

- **SQLite Engine**: Configured with `_journal_mode=WAL` for high concurrency.
- **Primary Table (`memories`)**: The source of truth. Uses a composite index (`idx_context_relevance`) for sub-millisecond contextual recall.
- **FTS5 Virtual Table (`memories_fts`)**: An inverted index synchronized via `AFTER` triggers to provide $O(\log n)$ keyword search performance.

## Performance

Echo is designed with performance in mind (P99 < 10ms) to ensure it does not bottleneck AI reasoning loops.

The following benchmarks were recorded on the current production environment:

- **CPU:** AMD Ryzen 7 7840HS (8C/16T) @ 3.8GHz
- **Storage:** NVMe SSD

### Current Performance (1,000+ Records)

By leveraging an FTS5 inverted index and optimized composite indexing, Echo achieves sub-millisecond latency for all read operations.

| Operation | Complexity | Latency (ms) | Note |
| :--- | :--- | :--- | :--- |
| **Recall** | $O(\log n)$ | **0.12 ms** | Indexed via `idx_context_relevance` |
| **Search** | $O(\log n)$ | **0.63 ms** | FTS5 Inverted Index |
| **Store** | $O(1)$ | **0.16 ms** | SQLite WAL UPSERT |
| **Delete** | $O(1)$ | **0.12 ms** | Record removal & FTS5 sync |

These metrics are formally verified via the Go benchmarking suite (`make bench`).
