# ADR 002: Implement FTS5 Inverted Index for Keyword Search

- **Status:** Accepted
- **Date:** 2026-02-28
- **Author:** Victoria Cheng

## Context and Problem Statement

The initial implementation of the `search_memories` MCP tool utilized a standard `LIKE %query%` SQL operator. Formal Go benchmarks revealed that this $O(n)$ full-table scan took ~9.20ms for a dataset of 1,000 records. As the AI's "brain" scales to 10,000+ memories, this linear approach risked violating the project's sub-10ms performance guarantee.

## Decision Outcome

Implemented a SQLite FTS5 (Full-Text Search) virtual table (`memories_fts`) synchronized with the primary `memories` table via automated `AFTER INSERT/UPDATE/DELETE` triggers. To maintain high-quality substring matching for very short queries, a "Hybrid Search" logic was adopted: queries < 3 characters fall back to `LIKE`, while longer queries utilize the FTS5 index with prefix matching (`*`).

## Consequences

### Positive

- **Architectural Scalability**: Shifted search complexity from $O(n)$ to $O(\log n)$.
- **Massive Performance Gain**: Achieved a 230x speedup on legacy hardware (Intel i7-4700HQ).

  | Operation | Baseline (LIKE) | Optimized (FTS5) | Improvement |
  | :--- | :--- | :--- | :--- |
  | **SearchMemories** | 9.20 ms | **0.04 ms** | **230x Faster** |

- **Data Integrity**: Automated synchronization triggers eliminate the risk of index drift without requiring an application-layer cron job.

### Negative

- **Build Complexity**: Requires the `sqlite_fts5` CGO build tag, which necessitates a slightly more complex `Makefile` (`GO_TAGS`) and requires developers to have FTS5-enabled SQLite in their Nix environment (`sqlite.interactive`).

## Verification

- [x] **Manual Check:** Verified the FTS index successfully backfilled existing data via the `rebuild` command.
- [x] **Automated Tests:** `BenchmarkSearchMemories` confirmed the 0.04ms latency. Total test suite maintains >80% coverage.
