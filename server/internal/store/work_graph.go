package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"taskline_server/api/model"
)

type runNodeExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

const runNodeSelectColumns = `
	id,run_id,node_key,title,capability,kind,position,depends_on,status,attempt,
	summary,next_step,artifact_ids,evidence,input_fingerprint,started_at,
	completed_at,updated_at`

func insertRunNode(
	ctx context.Context,
	execer runNodeExecer,
	runID string,
	node *model.RunNode,
	timestamp int64,
) error {
	if node == nil {
		return errors.New("run node required")
	}
	node.ID = newID()
	node.RunID = runID
	node.UpdatedAt = timestamp
	dependsOn, err := json.Marshal(node.DependsOn)
	if err != nil {
		return err
	}
	artifactIDs, err := json.Marshal(node.ArtifactIDs)
	if err != nil {
		return err
	}
	_, err = execer.ExecContext(ctx, `
		INSERT INTO run_nodes(
			id,run_id,node_key,title,capability,kind,position,depends_on,status,
			attempt,summary,next_step,artifact_ids,evidence,input_fingerprint,
			started_at,completed_at,updated_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		node.ID, node.RunID, node.Key, node.Title, node.Capability, node.Kind,
		node.Position, string(dependsOn), node.Status, node.Attempt, node.Summary,
		node.NextStep, string(artifactIDs), node.Evidence, node.InputFingerprint,
		node.StartedAt, node.CompletedAt, node.UpdatedAt,
	)
	return err
}

type runNodeScanner interface {
	Scan(...any) error
}

func scanRunNode(scanner runNodeScanner) (*model.RunNode, error) {
	var node model.RunNode
	var dependsOn, artifactIDs string
	err := scanner.Scan(
		&node.ID, &node.RunID, &node.Key, &node.Title, &node.Capability,
		&node.Kind, &node.Position, &dependsOn, &node.Status, &node.Attempt,
		&node.Summary, &node.NextStep, &artifactIDs, &node.Evidence,
		&node.InputFingerprint, &node.StartedAt, &node.CompletedAt, &node.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(dependsOn), &node.DependsOn); err != nil {
		return nil, fmt.Errorf("decode run node dependencies: %w", err)
	}
	if node.DependsOn == nil {
		node.DependsOn = []string{}
	}
	if err := json.Unmarshal([]byte(artifactIDs), &node.ArtifactIDs); err != nil {
		return nil, fmt.Errorf("decode run node artifacts: %w", err)
	}
	if node.ArtifactIDs == nil {
		node.ArtifactIDs = []string{}
	}
	return &node, nil
}

func (s *Store) GetRunNode(
	ctx context.Context,
	runID, nodeKey string,
) (*model.RunNode, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT `+runNodeSelectColumns+` FROM run_nodes WHERE run_id=? AND node_key=?`,
		runID,
		nodeKey,
	)
	return scanRunNode(row)
}

func (s *Store) ListRunNodes(ctx context.Context, runID string) ([]*model.RunNode, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+runNodeSelectColumns+` FROM run_nodes WHERE run_id=? ORDER BY position,id`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []*model.RunNode
	for rows.Next() {
		node, err := scanRunNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

func (s *Store) UpdateRunNode(
	ctx context.Context,
	node *model.RunNode,
) (*model.RunNode, error) {
	if node == nil {
		return nil, errors.New("run node required")
	}
	node.UpdatedAt = now()
	artifactIDs, err := json.Marshal(node.ArtifactIDs)
	if err != nil {
		return nil, err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE run_nodes
		   SET status=?,attempt=?,summary=?,next_step=?,artifact_ids=?,evidence=?,
		       input_fingerprint=?,started_at=?,completed_at=?,updated_at=?
		 WHERE id=?`,
		node.Status, node.Attempt, node.Summary, node.NextStep,
		string(artifactIDs), node.Evidence, node.InputFingerprint,
		node.StartedAt, node.CompletedAt, node.UpdatedAt, node.ID,
	)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, ErrNotFound
	}
	return s.GetRunNode(ctx, node.RunID, node.Key)
}

const runInterruptSelectColumns = `
	id,run_id,node_key,kind,prompt,options,status,response,requested_by,
	responded_by,created_at,resolved_at`

type runInterruptScanner interface {
	Scan(...any) error
}

func scanRunInterrupt(scanner runInterruptScanner) (*model.RunInterrupt, error) {
	var interrupt model.RunInterrupt
	var options string
	err := scanner.Scan(
		&interrupt.ID, &interrupt.RunID, &interrupt.NodeKey, &interrupt.Kind,
		&interrupt.Prompt, &options, &interrupt.Status, &interrupt.Response,
		&interrupt.RequestedBy, &interrupt.RespondedBy, &interrupt.CreatedAt,
		&interrupt.ResolvedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(options), &interrupt.Options); err != nil {
		return nil, fmt.Errorf("decode run interrupt options: %w", err)
	}
	if interrupt.Options == nil {
		interrupt.Options = []string{}
	}
	return &interrupt, nil
}

func (s *Store) CreateRunInterrupt(
	ctx context.Context,
	interrupt *model.RunInterrupt,
) (*model.RunInterrupt, error) {
	if interrupt == nil {
		return nil, errors.New("run interrupt required")
	}
	interrupt.ID = newID()
	interrupt.CreatedAt = now()
	interrupt.Status = model.RunInterruptPending
	options, err := json.Marshal(interrupt.Options)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO run_interrupts(
			id,run_id,node_key,kind,prompt,options,status,response,requested_by,
			responded_by,created_at,resolved_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		interrupt.ID, interrupt.RunID, interrupt.NodeKey, interrupt.Kind,
		interrupt.Prompt, string(options), interrupt.Status, interrupt.Response,
		interrupt.RequestedBy, interrupt.RespondedBy, interrupt.CreatedAt,
		interrupt.ResolvedAt,
	)
	if isFKErr(err) {
		return nil, ErrNotFound
	}
	if isUniqueErr(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	return s.GetRunInterrupt(ctx, interrupt.ID)
}

func (s *Store) GetRunInterrupt(
	ctx context.Context,
	id string,
) (*model.RunInterrupt, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT `+runInterruptSelectColumns+` FROM run_interrupts WHERE id=?`,
		id,
	)
	return scanRunInterrupt(row)
}

func (s *Store) ListPendingRunInterrupts(
	ctx context.Context,
	runID string,
) ([]*model.RunInterrupt, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT `+runInterruptSelectColumns+
			` FROM run_interrupts WHERE run_id=? AND status='pending' ORDER BY created_at,id`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var interrupts []*model.RunInterrupt
	for rows.Next() {
		interrupt, err := scanRunInterrupt(rows)
		if err != nil {
			return nil, err
		}
		interrupts = append(interrupts, interrupt)
	}
	return interrupts, rows.Err()
}

func (s *Store) HasAnsweredRunInterruptSince(
	ctx context.Context,
	runID, nodeKey string,
	createdAt int64,
) (bool, error) {
	var count int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM run_interrupts
		  WHERE run_id=? AND node_key=? AND status='answered' AND created_at>=?`,
		runID,
		nodeKey,
		createdAt,
	).Scan(&count)
	return count > 0, err
}

func (s *Store) UpdateRunInterrupt(
	ctx context.Context,
	interrupt *model.RunInterrupt,
) (*model.RunInterrupt, error) {
	if interrupt == nil {
		return nil, errors.New("run interrupt required")
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE run_interrupts
		   SET status=?,response=?,responded_by=?,resolved_at=?
		 WHERE id=? AND status='pending'`,
		interrupt.Status, interrupt.Response, interrupt.RespondedBy,
		interrupt.ResolvedAt, interrupt.ID,
	)
	if err != nil {
		return nil, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		if _, getErr := s.GetRunInterrupt(ctx, interrupt.ID); getErr != nil {
			return nil, getErr
		}
		return nil, ErrConflict
	}
	return s.GetRunInterrupt(ctx, interrupt.ID)
}
