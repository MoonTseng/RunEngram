CREATE UNIQUE INDEX idx_run_interrupts_pending_node
    ON run_interrupts(run_id, node_key)
    WHERE status = 'pending';
