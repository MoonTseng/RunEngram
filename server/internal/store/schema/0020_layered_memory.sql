-- 0020_layered_memory.sql · separate project rules from scoped experience.
--
-- Keep this migration text identical in server/migrations/ and
-- server/internal/store/schema/.

ALTER TABLE exploration_capsules
    ADD COLUMN memory_class TEXT NOT NULL DEFAULT 'experience'
        CHECK (memory_class IN ('experience','project-rule'));

ALTER TABLE exploration_capsules
    ADD COLUMN trigger TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_capsules_project_class_status
    ON exploration_capsules(project_id, memory_class, status, updated_at DESC);
