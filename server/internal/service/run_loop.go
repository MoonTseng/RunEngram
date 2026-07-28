package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"taskline_server/api/model"
	"taskline_server/internal/store"
)

const maxResumeEvents = 20

type StartRunInput struct {
	TaskID    string
	AgentName string
	AgentTool model.AgentTool
}

type SaveRunCheckpointInput struct {
	RunID     string
	AgentName string
	Status    model.RunStatus
	Summary   string
	NextStep  string
}

type RecordRunEventInput struct {
	RunID     string
	AgentName string
	Kind      model.RunEventKind
	Summary   string
	Details   map[string]any
}

type FinishRunInput struct {
	RunID     string
	AgentName string
	Status    model.RunStatus
	Summary   string
}

func (s *Service) StartOrResumeRun(
	ctx context.Context,
	input StartRunInput,
) (*model.AgentRun, bool, error) {
	input.TaskID = strings.TrimSpace(input.TaskID)
	input.AgentName = strings.TrimSpace(input.AgentName)
	if input.TaskID == "" {
		return nil, false, errors.New("task id required")
	}
	if !input.AgentTool.Valid() {
		return nil, false, fmt.Errorf("invalid agent tool %q", input.AgentTool)
	}
	task, err := s.st.GetTask(ctx, input.TaskID)
	if err != nil {
		return nil, false, err
	}
	if err := requireLiveOwner(task, input.AgentName); err != nil {
		return nil, false, err
	}
	active, err := s.st.GetActiveAgentRun(ctx, task.ID)
	if err == nil {
		if active.AgentName != input.AgentName {
			return nil, false, fmt.Errorf(
				"%w: active run belongs to %s",
				store.ErrConflict,
				active.AgentName,
			)
		}
		active.AgentTool = input.AgentTool
		active.Status = model.RunStatusRunning
		active, err = s.st.UpdateAgentRun(ctx, active)
		if err != nil {
			return nil, false, err
		}
		if _, err := s.appendTaskEvent(
			ctx,
			task.ID,
			string(model.RunEventResumed),
			"Resumed Agent run",
			runEventDetails(active, nil),
			active.UpdatedAt,
		); err != nil {
			return nil, false, err
		}
		return active, true, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, false, err
	}
	run, err := s.st.CreateAgentRun(ctx, &model.AgentRun{
		TaskID: task.ID, ProjectID: task.ProjectID, AgentName: input.AgentName,
		AgentTool: input.AgentTool, Status: model.RunStatusRunning,
	})
	if err != nil {
		return nil, false, err
	}
	if _, err := s.appendTaskEvent(
		ctx,
		task.ID,
		string(model.RunEventStarted),
		"Started Agent run",
		runEventDetails(run, nil),
		run.StartedAt,
	); err != nil {
		return nil, false, err
	}
	return run, false, nil
}

func (s *Service) SaveRunCheckpoint(
	ctx context.Context,
	input SaveRunCheckpointInput,
) (*model.AgentRun, error) {
	input.Summary = strings.TrimSpace(input.Summary)
	input.NextStep = strings.TrimSpace(input.NextStep)
	if input.Summary == "" {
		return nil, errors.New("checkpoint summary required")
	}
	if input.Status == "" {
		input.Status = model.RunStatusRunning
	}
	if input.Status != model.RunStatusRunning && input.Status != model.RunStatusBlocked {
		return nil, errors.New("checkpoint status must be running or blocked")
	}
	run, task, err := s.ownedRun(ctx, input.RunID, input.AgentName)
	if err != nil {
		return nil, err
	}
	if run.Status.Terminal() {
		return nil, fmt.Errorf("%w: run already finished", store.ErrConflict)
	}
	run.Status = input.Status
	run.Summary = input.Summary
	run.NextStep = input.NextStep
	run, err = s.st.UpdateAgentRun(ctx, run)
	if err != nil {
		return nil, err
	}
	kind := model.RunEventCheckpointSaved
	eventSummary := "Saved Agent checkpoint"
	if input.Status == model.RunStatusBlocked {
		kind = model.RunEventBlocked
		eventSummary = "Agent run blocked"
	}
	if _, err := s.appendTaskEvent(
		ctx,
		task.ID,
		string(kind),
		eventSummary,
		runEventDetails(run, map[string]any{
			"summary": run.Summary, "next_step": run.NextStep,
		}),
		run.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *Service) RecordRunEvent(
	ctx context.Context,
	input RecordRunEventInput,
) (*model.RunEvent, error) {
	input.Summary = strings.TrimSpace(input.Summary)
	if input.Summary == "" {
		return nil, errors.New("run event summary required")
	}
	switch input.Kind {
	case model.RunEventToolCalled,
		model.RunEventVerificationPassed,
		model.RunEventLearningDiscovered:
	default:
		return nil, fmt.Errorf("run event kind %q must use its lifecycle command", input.Kind)
	}
	run, task, err := s.ownedRun(ctx, input.RunID, input.AgentName)
	if err != nil {
		return nil, err
	}
	if run.Status.Terminal() {
		return nil, fmt.Errorf("%w: run already finished", store.ErrConflict)
	}
	details := runEventDetails(run, input.Details)
	if input.Kind == model.RunEventLearningDiscovered {
		note, err := s.captureRunLearning(ctx, run, task, details)
		if err != nil {
			return nil, err
		}
		details["learning_note_id"] = note.ID
	}
	taskEvent, err := s.appendTaskEvent(
		ctx,
		task.ID,
		string(input.Kind),
		input.Summary,
		details,
		nowMillis(),
	)
	if err != nil {
		return nil, err
	}
	return &model.RunEvent{
		ID: taskEvent.ID, RunID: run.ID, TaskID: task.ID,
		Actor: taskEvent.Actor, Kind: input.Kind, Summary: taskEvent.Summary,
		Details: taskEvent.Details, CreatedAt: taskEvent.CreatedAt,
	}, nil
}

func (s *Service) FinishRun(
	ctx context.Context,
	input FinishRunInput,
) (*model.AgentRun, error) {
	input.Summary = strings.TrimSpace(input.Summary)
	if input.Summary == "" {
		return nil, errors.New("run completion summary required")
	}
	if input.Status != model.RunStatusCompleted && input.Status != model.RunStatusFailed {
		return nil, errors.New("finish status must be completed or failed")
	}
	run, task, err := s.ownedRun(ctx, input.RunID, input.AgentName)
	if err != nil {
		return nil, err
	}
	if run.Status.Terminal() {
		return run, nil
	}
	run.Status = input.Status
	run.Summary = input.Summary
	run.NextStep = ""
	run.CompletedAt = nowMillis()
	run, err = s.st.UpdateAgentRun(ctx, run)
	if err != nil {
		return nil, err
	}
	kind := model.RunEventCompleted
	eventSummary := "Completed Agent run"
	if run.Status == model.RunStatusFailed {
		kind = model.RunEventFailed
		eventSummary = "Agent run failed"
	}
	if _, err := s.appendTaskEvent(
		ctx,
		task.ID,
		string(kind),
		eventSummary,
		runEventDetails(run, map[string]any{"summary": run.Summary}),
		run.CompletedAt,
	); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *Service) GetAgentRun(ctx context.Context, id string) (*model.AgentRun, error) {
	return s.st.GetAgentRun(ctx, id)
}

func (s *Service) ListAgentRuns(ctx context.Context, taskID string) ([]model.AgentRun, error) {
	if _, err := s.st.GetTask(ctx, taskID); err != nil {
		return nil, err
	}
	return s.st.ListAgentRuns(ctx, taskID)
}

func (s *Service) GetTaskResumeContext(
	ctx context.Context,
	taskID, agentName string,
) (*model.TaskResumeContext, error) {
	task, err := s.st.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if err := requireLiveOwner(task, agentName); err != nil {
		return nil, err
	}
	snapshot, err := s.GetOrCreateTaskContext(ctx, taskID)
	if err != nil {
		return nil, err
	}
	latest, err := s.st.GetLatestAgentRun(ctx, taskID)
	if errors.Is(err, store.ErrNotFound) {
		latest = nil
	} else if err != nil {
		return nil, err
	}
	events, err := s.ListTaskEvents(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if len(events) > maxResumeEvents {
		events = events[:maxResumeEvents]
	}
	return &model.TaskResumeContext{
		Snapshot: *snapshot, LatestRun: latest, RecentEvents: events,
	}, nil
}

func (s *Service) ownedRun(
	ctx context.Context,
	runID, agentName string,
) (*model.AgentRun, *model.Task, error) {
	run, err := s.st.GetAgentRun(ctx, strings.TrimSpace(runID))
	if err != nil {
		return nil, nil, err
	}
	task, err := s.st.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, nil, err
	}
	if err := requireLiveOwner(task, agentName); err != nil {
		return nil, nil, err
	}
	if run.AgentName != strings.TrimSpace(agentName) {
		return nil, nil, fmt.Errorf("%w: run belongs to %s", store.ErrConflict, run.AgentName)
	}
	return run, task, nil
}

func (s *Service) captureRunLearning(
	ctx context.Context,
	run *model.AgentRun,
	task *model.Task,
	details map[string]any,
) (*model.LearningNote, error) {
	trigger := detailString(details, "trigger")
	guidance := detailString(details, "guidance")
	if trigger == "" || guidance == "" {
		return nil, errors.New("learning.discovered requires details.trigger and details.guidance")
	}
	kind := model.LearningNoteAgentRecovery
	if raw := detailString(details, "kind"); raw != "" {
		kind = model.LearningNoteKind(raw)
	}
	return s.CaptureLearningNote(ctx, CaptureLearningNoteInput{
		ProjectID: task.ProjectID, SourceTaskID: task.ID, AgentName: run.AgentName,
		Kind: kind, Trigger: trigger, Guidance: guidance,
		Scope: detailString(details, "scope"), Producer: string(run.AgentTool),
	})
}

func runEventDetails(run *model.AgentRun, extra map[string]any) map[string]any {
	details := make(map[string]any, len(extra)+3)
	for key, value := range extra {
		details[key] = value
	}
	details["run_id"] = run.ID
	details["agent_name"] = run.AgentName
	details["agent_tool"] = run.AgentTool
	return details
}

func detailString(details map[string]any, key string) string {
	value, _ := details[key].(string)
	return strings.TrimSpace(value)
}
