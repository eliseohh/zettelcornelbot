-- Protocolo: SQLite solo guarda metadatos (Fuente de verdad = Markdown)

CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,           -- Normalmente el path relativo o un ID extraído del Frontmatter
    path TEXT NOT NULL UNIQUE,     -- Path relativo al root del vault
    hash TEXT NOT NULL,            -- Checksum del contenido para detectar cambios
    last_mod INTEGER NOT NULL,     -- Timestamp de modificación del archivo fs
    title TEXT,                    -- Cached title for fast search
    indexed_at INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE TABLE IF NOT EXISTS edges (
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    type TEXT DEFAULT 'wiki_link', -- 'wiki_link', 'parent', 'related'
    PRIMARY KEY (source_id, target_id, type),
    FOREIGN KEY(source_id) REFERENCES nodes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tags (
    node_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (node_id, tag),
    FOREIGN KEY(node_id) REFERENCES nodes(id) ON DELETE CASCADE
);

CREATE INDEX idx_nodes_title ON nodes(title);
CREATE INDEX idx_edges_target ON edges(target_id);
