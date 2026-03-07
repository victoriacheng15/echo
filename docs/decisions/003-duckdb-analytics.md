# ADR 003: Zero-Config DuckDB Analytics

- **Status:** Accepted
- **Date:** 2026-03-06
- **Author:** Victoria Cheng

## Context and Problem Statement

Echo requires a high-performance analytical layer to calculate FinOps (Cost-per-Project), GreenOps (Carbon Footprint), and Knowledge ROI (Hit Rate). Traditional OLTP databases like SQLite are not optimized for these complex aggregations and window functions. However, introducing a "sidecar" database (e.g., PostgreSQL, ClickHouse) violates our "Zero-Config" philosophy and complicates the user's development environment.

## Decision Outcome

We will utilize **DuckDB** as the primary analytical engine for Echo's telemetry and economics.

1. **Embedded OLAP:** DuckDB is an in-process, columnar database that requires no external server or configuration.
2. **CGO Integration:** We use the `duckdb-go/v2` driver with dynamic linking via Nix-provided libraries.
3. **Decoupled Telemetry:** To handle DuckDB's single-writer concurrency model, we implement an intermediate **JSONL Event Stream (`events.jsonl`)**.
   - The MCP server emits events asynchronously to the JSONL file.
   - The `AnalyticsService` periodically syncs this stream into DuckDB for analysis.
4. **Synthetic Baselines:** We implement "Synthetic Joule" and "Compute-per-Ms" calculations to provide immediate FinOps value without requiring hardware-level monitoring (Kepler).

## Consequences

### Positive

- **Performance**: Sub-millisecond analytical queries on millions of telemetry events using DuckDB's columnar engine.
- **Observability**: Native support for querying JSONL and Parquet files directly from the CLI without database locks.
- **Zero-Config**: The environment is fully declarative via `shell.nix`, requiring no manual database setup.
- **FinOps Standard**: Establishes a formal "Unit Economics" model for AI agent interactions.

### Negative

- **Concurrency**: DuckDB's single-writer lock prevents simultaneous access by the MCP server and the `duckdb` CLI.
- **Binary Size**: The CGO dependency on DuckDB increases the final binary size and build complexity.
- **Complexity**: Requires managing two distinct database lifecycles (SQLite for State, DuckDB for Analytics).

## Verification

- [x] **Manual Check:** Verified `analytics.duckdb` and `events.jsonl` creation in `~/.local/share/echo`.
- [x] **Automated Tests:** `internal/service/analytics_test.go` and `internal/service/refiner_test.go` passed.

### References

- [DuckDB Concurrency Model](https://duckdb.org/docs/stable/connect/concurrency)
- [FinOps Foundation: Unit Economics](https://www.finops.org/framework/capabilities/unit-economics/)
