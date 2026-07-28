CREATE TABLE exploration_capsules (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_task_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT '',
    evidence TEXT NOT NULL,
    labels TEXT NOT NULL DEFAULT '[]',
    fingerprints TEXT NOT NULL DEFAULT '[]',
    producer TEXT NOT NULL DEFAULT 'codex',
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active','stale','archived')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX idx_capsules_project_status
    ON exploration_capsules(project_id, status, updated_at DESC);

CREATE TABLE context_snapshots (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL UNIQUE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    payload TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE capsule_usages (
    id TEXT PRIMARY KEY,
    capsule_id TEXT NOT NULL REFERENCES exploration_capsules(id) ON DELETE CASCADE,
    task_id TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('used','helpful','rejected','stale')),
    notes TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE(capsule_id, task_id)
);

CREATE INDEX idx_capsule_usages_task ON capsule_usages(task_id);
