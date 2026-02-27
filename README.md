# Echo: Contextual Memory MCP Server

**Echo** is a persistent, contextual "brain" for AI agents. It functions as a Model Context Protocol (MCP) server that stores and recalls frequent sentences, architectural preferences, and project-specific snippets in a local SQLite database.

## Key Features

- **Contextual Recall**: Automatically provides relevant memories based on your current project or global preferences.
- **Relevance Ranking**: Memories are sorted by usage frequency and recency, ensuring the most important context stays at the top.
- **Privacy First**: All data is stored locally in a SQLite database. Nothing ever leaves your machine.
- **Lightweight & Portable**: Compiled into a single static Go binary.
