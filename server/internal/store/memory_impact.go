package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"taskline_server/api/model"
)

type MemoryImpactFilter struct {
	ProjectID string
	TaskID    string
	CapsuleID string
	States    []model.MemoryImpactState
	Limit     int
}

type MemoryImpactUpdate struct {
	State             *model.MemoryImpactState
	Stage             *string
	Notes             *string
	Evidence          *[]model.MemoryImpactEvidence
	Actor             *string
	ExpectedUpdatedAt int64
}

type memoryImpactScanner interface {
	Scan(dest ...any) error
}

func scanMemoryImpact(scanner memoryImpactScanner) (*model.MemoryImpact, error) {
	var impact model.MemoryImpact
	var reasonsJSON, evidenceJSON string
	var resolvedAt sql.NullInt64
	err := scanner.Scan(
		&impact.ID, &impact.ProjectID, &impact.TaskID, &impact.CapsuleID, &impact.State,
		&impact.RecallSource, &impact.ContextRevision, &impact.RecallScore, &reasonsJSON,
		&impact.Stage, &impact.Notes, &evidenceJSON, &impact.Actor,
		&impact.CreatedAt, &impact.UpdatedAt, &resolvedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(reasonsJSON), &impact.RecallReasons); err != nil {
		return nil, fmt.Errorf("decode memory impact reasons: %w", err)
	}
	if err := json.Unmarshal([]byte(evidenceJSON), &impact.Evidence); err != nil {
		return nil, fmt.Errorf("decode memory impact evidence: %w", err)
	}
	if impact.RecallReasons == nil {
		impact.RecallReasons = []string{}
	}
	if impact.Evidence == nil {
		impact.Evidence = []model.MemoryImpactEvidence{}
	}
	if resolvedAt.Valid {
		impact.ResolvedAt = resolvedAt.Int64
	}
	return &impact, nil
}

const memoryImpactColumns = `
	id,project_id,task_id,capsule_id,state,recall_source,context_revision,recall_score,
	recall_reasons_json,stage,notes,evidence_json,actor,created_at,updated_at,resolved_at`

func (s *Store) UpsertMemoryImpactRecall(ctx context.Context, impact *model.MemoryImpact) (*model.MemoryImpact, error) {
	if impact == nil {
		return nil, errors.New("memory impact required")
	}
	if impact.State == "" {
		impact.State = model.MemoryImpactRecalled
	}
	if impact.State != model.MemoryImpactRecalled {
		return nil, fmt.Errorf("recall upsert requires recalled state, got %q", impact.State)
	}
	reasonsJSON, err := json.Marshal(impact.RecallReasons)
	if err != nil {
		return nil, fmt.Errorf("encode memory impact reasons: %w", err)
	}
	nowMs := now()
	impact.ID = newID()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO memory_impacts(
			id,project_id,task_id,capsule_id,state,recall_source,context_revision,recall_score,
			recall_reasons_json,stage,notes,evidence_json,actor,created_at,updated_at,resolved_at
		) VALUES(?,?,?,?,?,?,?,?,?,'','','[]','',?,?,NULL)
		ON CONFLICT(task_id,capsule_id) DO UPDATE SET
			recall_source=excluded.recall_source,
			context_revision=excluded.context_revision,
			recall_score=excluded.recall_score,
			recall_reasons_json=excluded.recall_reasons_json,
			updated_at=excluded.updated_at`,
		impact.ID, impact.ProjectID, impact.TaskID, impact.CapsuleID, impact.State,
		strings.TrimSpace(impact.RecallSource), strings.TrimSpace(impact.ContextRevision),
		impact.RecallScore, string(reasonsJSON), nowMs, nowMs,
	)
	if isFKErr(err) {
		return nil, fmt.Errorf("%w: project %s", ErrNotFound, impact.ProjectID)
	}
	if err != nil {
		return nil, err
	}
	return s.getMemoryImpactByTaskCapsule(ctx, impact.TaskID, impact.CapsuleID)
}

func (s *Store) GetMemoryImpact(ctx context.Context, id string) (*model.MemoryImpact, error) {
	impact, err := scanMemoryImpact(s.db.QueryRowContext(ctx,
		`SELECT `+memoryImpactColumns+` FROM memory_impacts WHERE id=?`, strings.TrimSpace(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return impact, err
}

func (s *Store) getMemoryImpactByTaskCapsule(ctx context.Context, taskID, capsuleID string) (*model.MemoryImpact, error) {
	impact, err := scanMemoryImpact(s.db.QueryRowContext(ctx,
		`SELECT `+memoryImpactColumns+` FROM memory_impacts WHERE task_id=? AND capsule_id=?`,
		taskID, capsuleID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return impact, err
}

func (s *Store) ListMemoryImpacts(ctx context.Context, filter MemoryImpactFilter) ([]model.MemoryImpact, error) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 8)
	if projectID := strings.TrimSpace(filter.ProjectID); projectID != "" {
		clauses = append(clauses, "project_id=?")
		args = append(args, projectID)
	}
	if taskID := strings.TrimSpace(filter.TaskID); taskID != "" {
		clauses = append(clauses, "task_id=?")
		args = append(args, taskID)
	}
	if capsuleID := strings.TrimSpace(filter.CapsuleID); capsuleID != "" {
		clauses = append(clauses, "capsule_id=?")
		args = append(args, capsuleID)
	}
	if len(filter.States) > 0 {
		placeholders := make([]string, 0, len(filter.States))
		for _, state := range filter.States {
			if !state.Valid() {
				return nil, fmt.Errorf("invalid memory impact state %q", state)
			}
			placeholders = append(placeholders, "?")
			args = append(args, state)
		}
		clauses = append(clauses, "state IN ("+strings.Join(placeholders, ",")+")")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+memoryImpactColumns+` FROM memory_impacts WHERE `+
			strings.Join(clauses, " AND ")+` ORDER BY updated_at DESC,id LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	impacts := make([]model.MemoryImpact, 0)
	for rows.Next() {
		impact, err := scanMemoryImpact(rows)
		if err != nil {
			return nil, err
		}
		impacts = append(impacts, *impact)
	}
	return impacts, rows.Err()
}

func (s *Store) UpdateMemoryImpact(ctx context.Context, id string, update MemoryImpactUpdate) (*model.MemoryImpact, error) {
	current, err := s.GetMemoryImpact(ctx, id)
	if err != nil {
		return nil, err
	}
	if update.ExpectedUpdatedAt == 0 || current.UpdatedAt != update.ExpectedUpdatedAt {
		return nil, fmt.Errorf("%w: memory impact changed", ErrConflict)
	}
	if update.State != nil {
		current.State = *update.State
	}
	if update.Stage != nil {
		current.Stage = strings.TrimSpace(*update.Stage)
	}
	if update.Notes != nil {
		current.Notes = strings.TrimSpace(*update.Notes)
	}
	if update.Evidence != nil {
		current.Evidence = *update.Evidence
	}
	if update.Actor != nil {
		current.Actor = strings.TrimSpace(*update.Actor)
	}
	evidenceJSON, err := json.Marshal(current.Evidence)
	if err != nil {
		return nil, fmt.Errorf("encode memory impact evidence: %w", err)
	}
	nowMs := now()
	if nowMs <= current.UpdatedAt {
		nowMs = current.UpdatedAt + 1
	}
	var resolvedAt any
	if current.State.Terminal() {
		resolvedAt = nowMs
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE memory_impacts
		SET state=?,stage=?,notes=?,evidence_json=?,actor=?,updated_at=?,resolved_at=?
		WHERE id=? AND updated_at=?`,
		current.State, current.Stage, current.Notes, string(evidenceJSON), current.Actor,
		nowMs, resolvedAt, current.ID, update.ExpectedUpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, fmt.Errorf("%w: memory impact changed", ErrConflict)
	}
	return s.GetMemoryImpact(ctx, current.ID)
}

func (s *Store) MarkTaskMemoryImpactsUnconfirmed(ctx context.Context, taskID, actor string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE memory_impacts
		SET state='unconfirmed',actor=?,updated_at=?,resolved_at=NULL
		WHERE task_id=? AND state IN ('recalled','applied')`,
		strings.TrimSpace(actor), now(), strings.TrimSpace(taskID),
	)
	return err
}

func (s *Store) ListContextSnapshotsByProject(ctx context.Context, projectID string) ([]model.ContextSnapshot, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id,task_id,payload,created_at
		FROM context_snapshots
		WHERE project_id=?
		ORDER BY created_at,id`, strings.TrimSpace(projectID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	snapshots := make([]model.ContextSnapshot, 0)
	for rows.Next() {
		var snapshot model.ContextSnapshot
		var id, taskID, payload string
		var createdAt int64
		if err := rows.Scan(&id, &taskID, &payload, &createdAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(payload), &snapshot); err != nil {
			return nil, fmt.Errorf("decode context snapshot: %w", err)
		}
		snapshot.ID, snapshot.TaskID, snapshot.ProjectID, snapshot.CreatedAt =
			id, taskID, projectID, createdAt
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, rows.Err()
}

func (s *Store) GetMemoryImpactMetrics(ctx context.Context, projectID string) (*model.MemoryImpactMetrics, error) {
	var metrics model.MemoryImpactMetrics
	var appliedOrFinalTasks, confirmedTasks, snapshotTasks int
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(DISTINCT task_id),
			COUNT(*),
			COUNT(DISTINCT CASE WHEN state IN ('applied','helpful','rejected','stale') THEN task_id END),
			COUNT(DISTINCT CASE WHEN state='helpful' THEN task_id END),
			COUNT(CASE WHEN state='ignored' THEN 1 END),
			COUNT(CASE WHEN state='unconfirmed' THEN 1 END),
			COUNT(DISTINCT CASE WHEN state IN ('applied','helpful','rejected','stale') THEN task_id END),
			COUNT(DISTINCT CASE WHEN state IN ('helpful','rejected','stale') THEN task_id END)
		FROM memory_impacts
		WHERE project_id=?`, strings.TrimSpace(projectID),
	).Scan(
		&metrics.RecalledTaskCount, &metrics.RecalledMemoryCount,
		&metrics.AppliedTaskCount, &metrics.HelpfulTaskCount,
		&metrics.IgnoredCount, &metrics.UnconfirmedCount,
		&appliedOrFinalTasks, &confirmedTasks,
	)
	if err != nil {
		return nil, err
	}
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT task_id) FROM context_snapshots WHERE project_id=?`,
		strings.TrimSpace(projectID),
	).Scan(&snapshotTasks); err != nil {
		return nil, err
	}
	if snapshotTasks > 0 {
		metrics.RecallCoverageRate = float64(metrics.RecalledTaskCount) / float64(snapshotTasks)
	}
	if metrics.RecalledTaskCount > 0 {
		metrics.ApplicationRate = float64(appliedOrFinalTasks) / float64(metrics.RecalledTaskCount)
	}
	if appliedOrFinalTasks > 0 {
		metrics.ConfirmationRate = float64(confirmedTasks) / float64(appliedOrFinalTasks)
	}
	return &metrics, nil
}
