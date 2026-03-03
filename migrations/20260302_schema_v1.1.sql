-- migration_v1_1.sql: Migrates Echo database from v1.0 to v1.1.1
-- Includes: Refined Taxonomy mapping and manual source attribution to 'mcp'.

PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;

-- Create new table with updated schema
CREATE TABLE memories_v1_1_1 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content TEXT NOT NULL CHECK (length(content) > 0 AND length(content) <= 8192),
    context_key TEXT NOT NULL CHECK (length(context_key) > 0),
    entry_type TEXT DEFAULT 'directive' CHECK (entry_type IN ('directive', 'artifact', 'fact')),
    importance_score INTEGER DEFAULT 1 CHECK (importance_score BETWEEN 0 AND 10),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    source TEXT DEFAULT 'mcp' CHECK (source IN ('mcp', 'cli')),
    is_active BOOLEAN DEFAULT 1,
    tags TEXT CHECK (tags IS NULL OR json_valid(tags)),
    UNIQUE(content, context_key)
);

-- Migrate data with taxonomy mapping and manual source attribution
INSERT INTO memories_v1_1_1 (id, content, context_key, entry_type, importance_score, created_at, source, tags)
SELECT 
    id, content, context_key,
    CASE 
        WHEN entry_type = 'instruction' THEN 'directive'
        WHEN entry_type = 'snippet' THEN 'artifact'
        WHEN entry_type = 'request' THEN 'directive'
        WHEN entry_type = 'sentence' THEN 'fact'
        WHEN entry_type = 'boilerplate' THEN 'artifact'
        ELSE 'directive'
    END,
    CASE
        WHEN usage_count >= 10 THEN 10
        WHEN usage_count <= 1 THEN 1
        WHEN usage_count IS NULL THEN 1
        ELSE usage_count
    END,
    last_used,
    'mcp', -- Manually attribute all legacy data to 'mcp'
    -- Convert category to a JSON array in tags if valid JSON exists
    CASE 
        WHEN json_valid(metadata) AND json_extract(metadata, '$.category') IS NOT NULL 
        THEN json_array(json_extract(metadata, '$.category'))
        ELSE NULL
    END
FROM memories
WHERE id IS NOT NULL AND content != 'content'; -- Skip header row if present

-- Drop old table and rename new one
DROP TABLE memories;
ALTER TABLE memories_v1_1_1 RENAME TO memories;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_context_relevance ON memories(context_key, importance_score DESC);
CREATE INDEX IF NOT EXISTS idx_content ON memories(content);
CREATE INDEX IF NOT EXISTS idx_is_active ON memories(is_active);

COMMIT;
PRAGMA foreign_keys=ON;

-- Rebuild FTS triggers and index is handled by the application code on next start,
-- but for a clean slate, we can drop the old virtual table if it exists.
DROP TABLE IF EXISTS memories_fts;
