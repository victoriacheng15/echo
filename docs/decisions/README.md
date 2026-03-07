# Architecture Decision Records (ADR)

This directory contains the Architecture Decision Records for the Echo MCP Server. ADRs are used to document significant architectural and systemic choices, ensuring the "Why" behind a technical design is permanently recorded.

## Decision Log

| ADR | Title | Status |
| :--- | :--- | :--- |
| [003](003-duckdb-analytics.md) | Implement Zero-Config Analytics via DuckDB and Telemetry | 🔵 Accepted |
| [002](002-fts5-search.md) | Implement FTS5 Inverted Index for Keyword Search | 🔵 Accepted |
| [001](001-sqlite-wal.md) | Use SQLite with WAL Mode for Persistent Memory | 🔵 Accepted |

---

## 🛠️ Process & Standards

This section defines how we propose, evaluate, and document architectural changes.

### Decision Lifecycle

| Status | Meaning |
| :--- | :--- |
| **🟢 Proposed** | Planning phase. The design is being discussed or researched. |
| **🔵 Accepted** | Implementation phase or completed. This is the current project standard. |
| **🟡 Superseded** | Historical record. This decision has been replaced by a newer ADR. |

### Conventions

- **File Naming:** `00X-descriptive-title.md`
- **Dates:** Use ISO 8601 format (`YYYY-MM-DD`).
- **Formatting:** Use hyphens (`-`) for all lists; no numbered lists.

---

## Standard Headings (Template)

All new ADRs must adhere to the following markdown structure:

```markdown
# ADR [00X]: [Descriptive Title]

- **Status:** Proposed | Accepted | Superseded
- **Date:** YYYY-MM-DD
- **Author:** [Name]

## Context and Problem Statement
[What specific issue triggered this change? Provide the technical or business constraints.]

## Decision Outcome
[What was the chosen architectural path? Be explicit about the technology and the configuration.]

## Consequences

### Positive
- **[Benefit 1]**: [Description, ideally with empirical data/benchmarks]

### Negative
- **[Drawback 1]**: [Description of technical debt, complexity, or edge cases introduced]

## Verification
- [ ] **Manual Check:** [e.g., Verified logs/UI locally].
- [ ] **Automated Tests:** [e.g., benchmark suite passed].
```
