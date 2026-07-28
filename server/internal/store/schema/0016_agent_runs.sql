CREATE TABLE agent_runs (
    id           TEXT PRIMARY KEY,
    task_id      TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    project_id   TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    agent_name   TEXT NOT NULL,
    agent_tool   TEXT NOT NULL CHECK (agent_tool IN ('codex', 'claude-code', 'pi', 'other')),
    status       TEXT NOT NULL CHECK (status IN ('running', 'blocked', 'completed', 'failed')),
    summary      TEXT NOT NULL DEFAULT '',
    next_step    TEXT NOT NULL DEFAULT '',
    started_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL,
    completed_at INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_agent_runs_active_task
    ON agent_runs(task_id)
    WHERE status IN ('running', 'blocked');

CREATE INDEX idx_agent_runs_task_updated
    ON agent_runs(task_id, updated_at DESC);

CREATE INDEX idx_agent_runs_project_status
    ON agent_runs(project_id, status);
