ALTER TABLE agent_runs
    ADD COLUMN workflow_template TEXT NOT NULL DEFAULT 'single-loop'
        CHECK (workflow_template IN ('single-loop', 'cs-one-flow'));

ALTER TABLE agent_runs
    ADD COLUMN workflow_version INTEGER NOT NULL DEFAULT 1;

CREATE TABLE run_nodes (
    id                TEXT PRIMARY KEY,
    run_id            TEXT NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
    node_key          TEXT NOT NULL,
    title             TEXT NOT NULL,
    capability        TEXT NOT NULL,
    kind              TEXT NOT NULL,
    position          INTEGER NOT NULL,
    depends_on        TEXT NOT NULL DEFAULT '[]',
    status            TEXT NOT NULL
        CHECK (status IN ('pending', 'ready', 'running', 'waiting', 'completed', 'failed', 'skipped')),
    attempt           INTEGER NOT NULL DEFAULT 0,
    summary           TEXT NOT NULL DEFAULT '',
    next_step         TEXT NOT NULL DEFAULT '',
    artifact_ids      TEXT NOT NULL DEFAULT '[]',
    evidence          TEXT NOT NULL DEFAULT '',
    input_fingerprint TEXT NOT NULL DEFAULT '',
    started_at        INTEGER NOT NULL DEFAULT 0,
    completed_at      INTEGER NOT NULL DEFAULT 0,
    updated_at        INTEGER NOT NULL,
    UNIQUE(run_id, node_key)
);

CREATE INDEX idx_run_nodes_run_position
    ON run_nodes(run_id, position);

CREATE TABLE run_interrupts (
    id           TEXT PRIMARY KEY,
    run_id       TEXT NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
    node_key     TEXT NOT NULL,
    kind         TEXT NOT NULL
        CHECK (kind IN ('approval', 'question', 'choice', 'conflict')),
    prompt       TEXT NOT NULL,
    options      TEXT NOT NULL DEFAULT '[]',
    status       TEXT NOT NULL
        CHECK (status IN ('pending', 'answered', 'rejected')),
    response     TEXT NOT NULL DEFAULT '',
    requested_by TEXT NOT NULL DEFAULT '',
    responded_by TEXT NOT NULL DEFAULT '',
    created_at   INTEGER NOT NULL,
    resolved_at  INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_run_interrupts_run_status
    ON run_interrupts(run_id, status, created_at);
