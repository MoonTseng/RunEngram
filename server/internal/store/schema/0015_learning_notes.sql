CREATE TABLE learning_notes (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_task_id TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('human-correction','agent-recovery')),
    trigger TEXT NOT NULL,
    guidance TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT '',
    labels TEXT NOT NULL DEFAULT '[]',
    fingerprints TEXT NOT NULL DEFAULT '[]',
    producer TEXT NOT NULL DEFAULT 'codex',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','promoted','rejected')),
    evidence TEXT NOT NULL DEFAULT '',
    capsule_id TEXT NOT NULL DEFAULT '',
    rejection_reason TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    resolved_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_learning_notes_project_status
    ON learning_notes(project_id, status, updated_at DESC);
CREATE INDEX idx_learning_notes_task
    ON learning_notes(source_task_id, created_at DESC);
