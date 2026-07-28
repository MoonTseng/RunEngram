package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"taskline_server/api/model"
)

const agentRunSelectColumns = `
	id,task_id,project_id,agent_name,agent_tool,status,summary,next_step,
	started_at,updated_at,completed_at`

type agentRunScanner interface {
	Scan(dest ...any) error
}

func scanAgentRun(scanner agentRunScanner) (*model.AgentRun, error) {
	var run model.AgentRun
	err := scanner.Scan(
		&run.ID,
		&run.TaskID,
		&run.ProjectID,
		&run.AgentName,
		&run.AgentTool,
		&run.Status,
		&run.Summary,
		&run.NextStep,
		&run.StartedAt,
		&run.UpdatedAt,
		&run.CompletedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *Store) CreateAgentRun(ctx context.Context, run *model.AgentRun) (*model.AgentRun, error) {
	if run == nil {
		return nil, errors.New("agent run required")
	}
	run.ID = newID()
	run.StartedAt = now()
	run.UpdatedAt = run.StartedAt
	run.CompletedAt = 0
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_runs(
			id,task_id,project_id,agent_name,agent_tool,status,summary,next_step,
			started_at,updated_at,completed_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		run.ID,
		run.TaskID,
		run.ProjectID,
		run.AgentName,
		run.AgentTool,
		run.Status,
		run.Summary,
		run.NextStep,
		run.StartedAt,
		run.UpdatedAt,
		run.CompletedAt,
	)
	if isFKErr(err) {
		return nil, fmt.Errorf("%w: task %s", ErrNotFound, run.TaskID)
	}
	if isUniqueErr(err) {
		return nil, fmt.Errorf("%w: task %s already has an active run", ErrConflict, run.TaskID)
	}
	if err != nil {
		return nil, err
	}
	return s.GetAgentRun(ctx, run.ID)
}

func (s *Store) GetAgentRun(ctx context.Context, id string) (*model.AgentRun, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT `+agentRunSelectColumns+` FROM agent_runs WHERE id=?`,
		id,
	)
	return scanAgentRun(row)
}

func (s *Store) GetActiveAgentRun(ctx context.Context, taskID string) (*model.AgentRun, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT `+agentRunSelectColumns+`
		   FROM agent_runs
		  WHERE task_id=? AND status IN (?,?)
		  ORDER BY updated_at DESC LIMIT 1`,
		taskID,
		model.RunStatusRunning,
		model.RunStatusBlocked,
	)
	return scanAgentRun(row)
}

func (s *Store) GetLatestAgentRun(ctx context.Context, taskID string) (*model.AgentRun, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT `+agentRunSelectColumns+`
		   FROM agent_runs
		  WHERE task_id=?
		  ORDER BY updated_at DESC, rowid DESC LIMIT 1`,
		taskID,
	)
	return scanAgentRun(row)
}

func (s *Store) ListAgentRuns(ctx context.Context, taskID string) ([]model.AgentRun, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+agentRunSelectColumns+`
		   FROM agent_runs
		  WHERE task_id=?
		  ORDER BY updated_at DESC, rowid DESC`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := make([]model.AgentRun, 0)
	for rows.Next() {
		run, err := scanAgentRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, *run)
	}
	return runs, rows.Err()
}

func (s *Store) UpdateAgentRun(ctx context.Context, run *model.AgentRun) (*model.AgentRun, error) {
	if run == nil {
		return nil, errors.New("agent run required")
	}
	run.UpdatedAt = now()
	result, err := s.db.ExecContext(ctx, `
		UPDATE agent_runs
		   SET agent_tool=?,status=?,summary=?,next_step=?,updated_at=?,completed_at=?
		 WHERE id=?`,
		run.AgentTool,
		run.Status,
		run.Summary,
		run.NextStep,
		run.UpdatedAt,
		run.CompletedAt,
		run.ID,
	)
	if isUniqueErr(err) {
		return nil, fmt.Errorf("%w: task %s already has an active run", ErrConflict, run.TaskID)
	}
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, ErrNotFound
	}
	return s.GetAgentRun(ctx, run.ID)
}

type RunMetrics struct {
	RunCount          int
	CompletedRunCount int
	ActiveRunCount    int
	BlockedRunCount   int
	ResumedRunCount   int
	RecoveredRunCount int
}

func (s *Store) GetRunMetrics(ctx context.Context, projectID string) (*RunMetrics, error) {
	var metrics RunMetrics
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COUNT(CASE WHEN status='completed' THEN 1 END),
			COUNT(CASE WHEN status IN ('running','blocked') THEN 1 END)
		  FROM agent_runs
		 WHERE project_id=?`,
		projectID,
	).Scan(&metrics.RunCount, &metrics.CompletedRunCount, &metrics.ActiveRunCount)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(DISTINCT CASE WHEN e.action='run.blocked' THEN json_extract(e.details,'$.run_id') END),
			COUNT(DISTINCT CASE WHEN e.action='run.resumed' THEN json_extract(e.details,'$.run_id') END)
		  FROM task_events e
		  JOIN agent_runs r ON r.id=json_extract(e.details,'$.run_id')
		 WHERE r.project_id=?`,
		projectID,
	).Scan(&metrics.BlockedRunCount, &metrics.ResumedRunCount)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		  FROM agent_runs r
		 WHERE r.project_id=?
		   AND r.status='completed'
		   AND EXISTS (
				SELECT 1 FROM task_events e
				 WHERE e.action='run.blocked'
				   AND json_extract(e.details,'$.run_id')=r.id
		   )`,
		projectID,
	).Scan(&metrics.RecoveredRunCount)
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}
