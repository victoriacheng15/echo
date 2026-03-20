# ADR 005: Expand into a Dual-Interface Application (CLI + MCP)

- **Status:** Accepted
- **Date:** 2026-03-20
- **Author:** Victoria Cheng

## Context and Problem Statement

The system was initially designed as a background service accessible exclusively via the Model Context Protocol (MCP) for AI agents. While this architecture successfully provides agents with persistent memory, it limits user interaction and administrative oversight:

- **User Flexibility Constraint**: Users currently have no direct, first-class method to interact with, audit, or curate the SQLite "Brain" without using an AI intermediary or raw SQL queries.
- **Workflow Rigidity**: Some maintenance tasks (e.g., FTS5 index rebuilds, analytics sync) are better suited for a direct terminal-based interface than a conversational agent.

## Decision Outcome

The expansion of `echo` into a **Dual-Interface Application** is achieved through the implementation of a standard flag-based Command Line Interface (CLI). This architecture provides flexibility to choose the most appropriate interface for a given task:

1. **Shared Service Layer**: Both the MCP server and the CLI consume the same `internal/service` components (`MemoryService`, `AnalyticsService`, `RateService`), ensuring 100% state consistency and logic reuse.
2. **Unified Entry Point & Smart TTY Routing**: A "Fat Binary" pattern is implemented in `cmd/echo/main.go`. The application uses intelligent TTY detection to automatically switch between the interactive CLI (human) and the background JSON-RPC server (AI).
3. **Subcommand Dispatcher**: A new `internal/cli` package handles subcommand routing and flag parsing, keeping the primary entry point focused on interface multiplexing.
4. **Binary Naming & System Integration**: To prevent collisions with the standard system `echo` command, the binary is officially named `echo-cli` and standardized for installation in `/usr/local/bin`.
5. **Multi-Format Output**: The CLI supports `table`, `json`, and `csv` output formats to facilitate both human readability and automated data pipelines.
6. **Administrative Maintenance**: Maintenance routines such as `maintain --rebuild` (for SQLite FTS5) and `maintain --sync` (for DuckDB) are first-class citizens in the CLI.

## Consequences

### Positive

- **Operational Simplicity**: A single binary manages the entire memory lifecycle, reducing deployment complexity and ensuring interface parity.
- **Conflict Prevention**: Naming the tool `echo-cli` preserves system integrity while providing a clear namespace for memory management.
- **User Flexibility**: Provides users with the choice between AI-driven interactions (MCP) and direct terminal-based control (CLI).
- **Improved Curation**: Simplifies human-in-the-loop memory management, allowing for rapid auditing and correction of stored knowledge.

### Negative

- **Binary Size**: Including multiple interfaces and formatting logic slightly increases the final binary footprint.
- **Routing Complexity**: Requires robust TTY detection logic to ensure the AI host doesn't accidentally trigger CLI usage messages.

## Verification

- [x] **Architecture**: Implemented a unified entry point in `cmd/echo/main.go` with TTY-based multiplexing.
- [x] **Unit Tests**: Established a test suite in `internal/cli/dispatcher_test.go` using in-memory SQLite to verify all subcommands.
- [x] **Build Validation**: Confirmed successful compilation and installation of the `echo-cli` binary via the project's Makefile.
