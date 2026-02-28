# ADR 001: Use SQLite with WAL Mode for Persistent Memory

- **Status:** Accepted
- **Date:** 2026-02-27
- **Author:** Victoria Cheng

## Context and Problem Statement

Stateless LLM sessions reset context every time the process exits, forcing developers to manually re-establish project preferences and architectural instructions. The Echo MCP server required a fast, local, and concurrent storage mechanism to persist this "brain" across sessions. Furthermore, modern AI agents (like Claude Desktop) may execute multiple tool calls concurrently, requiring a database that can handle simultaneous reads and writes without locking errors.

## Decision Outcome

Chose SQLite configured with Write-Ahead Logging (WAL) mode (`_journal_mode=WAL`) and a busy timeout (`_busy_timeout=5000`) over standard rollback journaling or a heavier standalone Vector DB.

## Consequences

### Positive

- **High Concurrency**: WAL mode allows readers to operate simultaneously with a writer, eliminating `database is locked` errors during aggressive AI tool usage.
- **Extreme Performance**: Benchmarks show `StoreMemory` (UPSERT logic) completing in ~0.85ms on legacy hardware.
- **Portability**: The entire "brain" is contained in a single XDG-compliant `echo.db` file, requiring no external daemons.

### Negative

- **Test Artifact Pollution**: WAL mode creates `-shm` (shared memory) and `-wal` sidecar files during operation, requiring explicit `defer os.Remove()` cleanup logic in the Go testing suites to maintain repository hygiene.

## Verification

- [x] **Manual Check:** Verified the creation and persistence of `echo.db` in `~/.local/share/echo/`.
- [x] **Automated Tests:** Benchmarks successfully executed, proving sub-millisecond write latency (`make bench`).
