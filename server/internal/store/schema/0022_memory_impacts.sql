CREATE TABLE memory_impacts (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL,
  capsule_id TEXT NOT NULL,
  state TEXT NOT NULL CHECK (
    state IN ('recalled','applied','ignored','helpful','rejected','stale','unconfirmed')
  ),
  recall_source TEXT NOT NULL DEFAULT '',
  context_revision TEXT NOT NULL DEFAULT '',
  recall_score REAL NOT NULL DEFAULT 0,
  recall_reasons_json TEXT NOT NULL DEFAULT '[]',
  stage TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  evidence_json TEXT NOT NULL DEFAULT '[]',
  actor TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  resolved_at INTEGER,
  UNIQUE(task_id, capsule_id)
);

CREATE INDEX idx_memory_impacts_project_updated
  ON memory_impacts(project_id, updated_at DESC);
CREATE INDEX idx_memory_impacts_task
  ON memory_impacts(task_id, updated_at DESC);
CREATE INDEX idx_memory_impacts_capsule
  ON memory_impacts(capsule_id, updated_at DESC);
