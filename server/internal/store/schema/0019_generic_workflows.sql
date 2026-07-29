ALTER TABLE agent_runs
    ADD COLUMN workflow_key TEXT NOT NULL DEFAULT 'single-loop';

UPDATE agent_runs
   SET workflow_key = CASE workflow_template
       WHEN 'cs-one-flow' THEN 'engineering-flow'
       ELSE workflow_template
   END;
