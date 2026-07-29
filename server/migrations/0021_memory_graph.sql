-- 0021_memory_graph.sql · typed, inspectable engineering-memory relations.
--
-- Keep this migration text identical in server/migrations/ and
-- server/internal/store/schema/.

CREATE TABLE memory_relations (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_capsule_id TEXT NOT NULL REFERENCES exploration_capsules(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL
        CHECK (relation_type IN (
            'derived-from',
            'validated-by',
            'applies-to',
            'supersedes',
            'conflicts-with',
            'caused-by'
        )),
    target_kind TEXT NOT NULL
        CHECK (target_kind IN ('capsule','task','artifact','scope')),
    target_ref TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    UNIQUE(project_id, source_capsule_id, relation_type, target_kind, target_ref)
);

CREATE INDEX idx_memory_relations_source
    ON memory_relations(source_capsule_id, relation_type);

CREATE INDEX idx_memory_relations_target
    ON memory_relations(project_id, target_kind, target_ref, relation_type);
