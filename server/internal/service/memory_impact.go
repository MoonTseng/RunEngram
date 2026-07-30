package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"taskline_server/api/model"
	"taskline_server/internal/store"
)

type RecordMemoryImpactInput struct {
	ImpactID          string
	State             model.MemoryImpactState
	Stage             string
	Notes             string
	Evidence          []model.MemoryImpactEvidence
	Actor             string
	AgentName         string
	ExpectedUpdatedAt int64
}

var supportedMemoryEvidenceKinds = map[string]struct{}{
	"command":        {},
	"task-doc":       {},
	"task-event":     {},
	"link":           {},
	"code-reference": {},
	"observation":    {},
}

func (s *Service) ListMemoryImpacts(
	ctx context.Context,
	filter store.MemoryImpactFilter,
) ([]model.MemoryImpact, error) {
	return s.st.ListMemoryImpacts(ctx, filter)
}

func (s *Service) RecordCapsuleImpact(
	ctx context.Context,
	input RecordUsageInput,
	state model.MemoryImpactState,
	expectedUpdatedAt int64,
) (*model.MemoryImpact, error) {
	capsule, err := s.st.GetCapsule(ctx, strings.TrimSpace(input.CapsuleID))
	if err != nil {
		return nil, err
	}
	task, err := s.st.GetTask(ctx, strings.TrimSpace(input.TaskID))
	if err != nil {
		return nil, err
	}
	if capsule.ProjectID != task.ProjectID {
		return nil, fmt.Errorf("%w: capsule and task belong to different projects", store.ErrConflict)
	}
	impacts, err := s.st.ListMemoryImpacts(ctx, store.MemoryImpactFilter{
		TaskID: task.ID, CapsuleID: capsule.ID, Limit: 1,
	})
	if err != nil {
		return nil, err
	}
	var impact *model.MemoryImpact
	if len(impacts) == 0 {
		impact, err = s.st.UpsertMemoryImpactRecall(ctx, &model.MemoryImpact{
			ProjectID: task.ProjectID, TaskID: task.ID, CapsuleID: capsule.ID,
			State: model.MemoryImpactRecalled, RecallSource: "legacy-usage",
		})
		if err != nil {
			return nil, err
		}
	} else {
		impact = &impacts[0]
	}
	if expectedUpdatedAt == 0 {
		expectedUpdatedAt = impact.UpdatedAt
	}
	if state.Terminal() && len(input.Evidence) == 0 {
		input.Evidence = []model.MemoryImpactEvidence{{
			Kind: "observation", Ref: "legacy-capsule-usage",
			Summary: strings.TrimSpace(input.Notes),
		}}
	}
	return s.RecordMemoryImpact(ctx, RecordMemoryImpactInput{
		ImpactID: impact.ID, State: state, Stage: input.Stage,
		Notes: input.Notes, Evidence: input.Evidence, Actor: input.Actor,
		AgentName: input.AgentName, ExpectedUpdatedAt: expectedUpdatedAt,
	})
}

func (s *Service) RecordMemoryImpact(
	ctx context.Context,
	input RecordMemoryImpactInput,
) (*model.MemoryImpact, error) {
	if !input.State.Valid() {
		return nil, fmt.Errorf("invalid memory impact state %q", input.State)
	}
	current, err := s.st.GetMemoryImpact(ctx, strings.TrimSpace(input.ImpactID))
	if err != nil {
		return nil, err
	}
	input.Notes = strings.TrimSpace(input.Notes)
	input.Actor = strings.TrimSpace(input.Actor)
	input.AgentName = strings.TrimSpace(input.AgentName)
	if input.Actor == "" {
		input.Actor = input.AgentName
	}
	if input.State == model.MemoryImpactApplied || input.State == model.MemoryImpactIgnored ||
		input.State.Terminal() {
		if input.Notes == "" {
			return nil, errors.New("memory impact notes required")
		}
	}
	if input.State.Terminal() {
		if err := validateMemoryImpactEvidence(input.Evidence); err != nil {
			return nil, err
		}
	}
	if current.State.Terminal() && current.State != input.State {
		if input.AgentName != "" {
			return nil, errors.New("agent cannot overwrite a final memory impact")
		}
		if input.ExpectedUpdatedAt == 0 {
			return nil, errors.New("expected_updated_at required to correct a final memory impact")
		}
	} else if !current.State.CanTransitionTo(input.State) {
		return nil, fmt.Errorf("invalid memory impact transition %s -> %s", current.State, input.State)
	}
	stage := strings.TrimSpace(input.Stage)
	actor := input.Actor
	impact, err := s.st.UpdateMemoryImpact(ctx, current.ID, store.MemoryImpactUpdate{
		State:             &input.State,
		Stage:             &stage,
		Notes:             &input.Notes,
		Evidence:          &input.Evidence,
		Actor:             &actor,
		ExpectedUpdatedAt: input.ExpectedUpdatedAt,
	})
	if err != nil {
		return nil, err
	}
	if outcome, ok := memoryImpactOutcome(input.State); ok {
		if _, err := s.st.UpsertCapsuleUsage(ctx, &model.CapsuleUsage{
			CapsuleID: current.CapsuleID,
			TaskID:    current.TaskID,
			Outcome:   outcome,
			Notes:     input.Notes,
		}); err != nil {
			return nil, err
		}
	}
	if input.State == model.MemoryImpactStale {
		status := model.CapsuleStatusStale
		if _, err := s.st.UpdateCapsule(ctx, current.CapsuleID, store.CapsuleUpdate{Status: &status}); err != nil {
			return nil, err
		}
	}
	return impact, nil
}

func validateMemoryImpactEvidence(evidence []model.MemoryImpactEvidence) error {
	if len(evidence) == 0 {
		return errors.New("memory impact evidence required for final outcome")
	}
	for index := range evidence {
		evidence[index].Kind = strings.TrimSpace(evidence[index].Kind)
		evidence[index].Ref = strings.TrimSpace(evidence[index].Ref)
		evidence[index].Summary = strings.TrimSpace(evidence[index].Summary)
		if _, ok := supportedMemoryEvidenceKinds[evidence[index].Kind]; !ok {
			return fmt.Errorf("unsupported memory impact evidence kind %q", evidence[index].Kind)
		}
		if evidence[index].Ref == "" && evidence[index].Summary == "" {
			return fmt.Errorf("memory impact evidence %d requires ref or summary", index+1)
		}
	}
	return nil
}

func memoryImpactOutcome(state model.MemoryImpactState) (model.CapsuleOutcome, bool) {
	switch state {
	case model.MemoryImpactApplied:
		return model.CapsuleOutcomeUsed, true
	case model.MemoryImpactHelpful:
		return model.CapsuleOutcomeHelpful, true
	case model.MemoryImpactRejected:
		return model.CapsuleOutcomeRejected, true
	case model.MemoryImpactStale:
		return model.CapsuleOutcomeStale, true
	default:
		return "", false
	}
}

func (s *Service) recordRecallReceipts(
	ctx context.Context,
	task model.Task,
	recall model.MemoryRecall,
	source string,
) error {
	explanations := make(map[string]model.MemoryRecallExplanation, len(recall.Explanations))
	for _, explanation := range recall.Explanations {
		explanations[explanation.CapsuleID] = explanation
	}
	capsules := make([]model.ExplorationCapsule, 0, len(recall.ProjectRules)+len(recall.SuggestedCapsules))
	capsules = append(capsules, recall.ProjectRules...)
	capsules = append(capsules, recall.SuggestedCapsules...)
	seen := make(map[string]struct{}, len(capsules))
	for _, capsule := range capsules {
		if _, ok := seen[capsule.ID]; ok {
			continue
		}
		seen[capsule.ID] = struct{}{}
		explanation := explanations[capsule.ID]
		reasons := make([]string, 0, len(explanation.Reasons)+len(explanation.Warnings))
		for _, reason := range explanation.Reasons {
			value := strings.TrimSpace(reason.Value)
			if value == "" {
				reasons = append(reasons, reason.Code)
			} else {
				reasons = append(reasons, reason.Code+":"+value)
			}
		}
		for _, warning := range explanation.Warnings {
			reasons = append(reasons, "warning:"+strings.TrimSpace(warning))
		}
		if _, err := s.st.UpsertMemoryImpactRecall(ctx, &model.MemoryImpact{
			ProjectID:       task.ProjectID,
			TaskID:          task.ID,
			CapsuleID:       capsule.ID,
			State:           model.MemoryImpactRecalled,
			RecallSource:    source,
			ContextRevision: recall.ContextRevision,
			RecallScore:     explanation.Score,
			RecallReasons:   reasons,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ReconcileMemoryImpacts(ctx context.Context, projectID string) error {
	snapshots, err := s.st.ListContextSnapshotsByProject(ctx, strings.TrimSpace(projectID))
	if err != nil {
		return err
	}
	for _, snapshot := range snapshots {
		recall := model.MemoryRecall{
			TaskID:            snapshot.TaskID,
			ProjectID:         snapshot.ProjectID,
			ProjectRules:      snapshot.ProjectRules,
			SuggestedCapsules: snapshot.SuggestedCapsules,
			ContextRevision:   snapshot.ContextRevision,
			Explanations:      snapshot.Explanations,
		}
		if err := s.recordRecallReceipts(ctx, snapshot.Task, recall, "snapshot-backfill"); err != nil {
			return err
		}
	}
	return nil
}
