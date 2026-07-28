-- 0009_docs_task_type.sql · add 'docs' as a first-class task type.
-- SQLite cannot alter a CHECK constraint in place, so rebuild tasks while
-- preserving the v8 labels column and existing child-table relationships.
-- Keep this migration text identical in server/migrations/ and
-- server/internal/store/schema/: the former is external migration history,
-- and the latter is embedded into the binary for fresh database creation.

PRAGMA defer_foreign_keys = ON;

CREATE TEMP TABLE task_deps_0009 AS SELECT * FROM task_deps;
CREATE TEMP TABLE task_images_0009 AS SELECT * FROM task_images;
CREATE TEMP TABLE task_docs_0009 AS SELECT * FROM task_docs;
CREATE TEMP TABLE task_links_0009 AS SELECT * FROM task_links;

CREATE TABLE tasks_new (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL CHECK (type IN ('feature','bug','docs')),
    state       TEXT NOT NULL CHECK (state IN ('pending','start','spec','dev','test','review','done')),
    priority    INTEGER NOT NULL DEFAULT 0,
    labels      TEXT NOT NULL DEFAULT '[]',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

INSERT INTO tasks_new (id, project_id, title, description, type, state, priority, labels, created_at, updated_at)
    SELECT id, project_id, title, description, type, state, priority, labels, created_at, updated_at
      FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

INSERT INTO task_deps SELECT * FROM task_deps_0009;
INSERT INTO task_images SELECT * FROM task_images_0009;
INSERT INTO task_docs SELECT * FROM task_docs_0009;
INSERT INTO task_links SELECT * FROM task_links_0009;

DROP TABLE task_deps_0009;
DROP TABLE task_images_0009;
DROP TABLE task_docs_0009;
DROP TABLE task_links_0009;

CREATE INDEX idx_tasks_project_state ON tasks(project_id, state);
CREATE INDEX idx_tasks_priority      ON tasks(project_id, priority DESC);
