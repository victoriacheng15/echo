# Echo Memory Taxonomy & Governance

To ensure the "Brain" remains high-signal and searchable, all stored memories MUST adhere to these taxonomy and operational rules.

## 1. Intent-Based Taxonomy (Strict Enum)

You MUST classify every entry into one of the following `entry_type` values:

| Type | Intent | Example Use Case |
| :--- | :--- | :--- |
| `directive` | Mandates, user preferences, and operational rules. | "Always use tabs," "Prefer Go for new CLIs." |
| `artifact` | Reusable code, templates, and structural fragments. | A `docker-compose.yml` block or a `commit.md` template. |
| `fact` | Purely observational truths and architectural state. | "The SQLite database is stored in ~/.local/share." |

## 2. Context Scoping (Naming Convention)

You MUST use lower-case, hyphenated strings for `context_key`. Follow these patterns:

- **Global:** Use `global` for memories that apply across all projects.
- **Project:** Use `project:<name>` (e.g., `project:echo`) for project-specific logic.
- **Feature/Domain:** Use `type:identifier` (e.g., `auth:jwt`) for granular scoping.

## 3. Operational Guidelines

1. **Search Before Store:** Always run `search_memories` first to prevent duplicate entries.
2. **Reinforcement:** Use `store_memory` with an existing `content` and `context_key` to reinforce an entry (it triggers an `ON CONFLICT` importance reinforcement).
3. **Surgical Updates:** Use `update_memory` with a specific `id` to refine the description (text) of an existing memory without losing its importance score or history.
4. **Three-Stage Deletion:** To delete a memory, you MUST:
    - a. Use `search_for_deletion` to retrieve the exact memory.
    - b. Display the memory to the user and obtain explicit confirmation.
    - c. Use `delete_memory` ONLY after the user confirms.
4. **Chunking:** Keep each memory focused on a single concept (max 8KB). If content is larger, split it into logical parts (e.g., `part-1`).
5. **Categorization:** Use the `tags` array for cross-context indexing (e.g., `["security", "go-standard"]\`).
