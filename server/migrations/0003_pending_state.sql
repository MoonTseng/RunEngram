-- 0003_pending_state.sql · introduce 'pending' as a non-runnable parking
-- lot and rename the entry state 'created' → 'start'. SQLite has no
-- ALTER TABLE for changing a CHECK constraint, so the supported recipe
-- is CREATE NEW / COPY / DROP OLD / RENAME. We rely on
-- PRAGMA defer_foreign_keys = ON to let DROP TABLE tasks coexist briefly
-- with task_deps' FK references inside this transaction — the FK
-- definitions in task_deps / task_images are not destroyed, they re-bind
-- to the renamed `tasks` table at commit time. Verified with
-- `PRAGMA foreign_key_check` + cascade-delete after the swap.
--
-- The rename created → start happens inside the INSERT (CASE WHEN) and
-- NOT via a prior UPDATE on `tasks`: the old CHECK constraint forbids
-- the value 'start', so an UPDATE would fail on any database that
-- actually has rows in state='created'.

PRAGMA defer_foreign_keys = ON;

CREATE TABLE tasks_new (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL CHECK (type IN ('feature','bug')),
    state       TEXT NOT NULL CHECK (state IN ('pending','start','design','dev','review','done')),
    priority    INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

INSERT INTO tasks_new (id, project_id, title, description, type, state, priority, created_at, updated_at)
    SELECT id, project_id, title, description, type,
           CASE WHEN state = 'created' THEN 'start' ELSE state END,
           priority, created_at, updated_at
      FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

CREATE INDEX idx_tasks_project_state ON tasks(project_id, state);
CREATE INDEX idx_tasks_priority      ON tasks(project_id, priority DESC);
