package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"taskline_server/api/model"
	"taskline_server/internal/service"
	"taskline_server/internal/store"
)

func TestAgentRunCheckpointsResumeAndMeasureRecovery(t *testing.T) {
	ctx := service.WithActor(context.Background(), "codex")
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "run-loop", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(
		ctx,
		project.ID,
		"Migrate URL service",
		"Move callers, verify behavior, remove compatibility service",
		model.TaskTypeFeature,
		2,
		true,
		[]string{"webview"},
	)
	require.NoError(t, err)
	task, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)

	run, resumed, err := svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "codex", AgentTool: model.AgentToolCodex,
	})
	require.NoError(t, err)
	require.False(t, resumed)
	require.Equal(t, model.RunStatusRunning, run.Status)

	run, err = svc.SaveRunCheckpoint(ctx, service.SaveRunCheckpointInput{
		RunID: run.ID, AgentName: "codex", Status: model.RunStatusBlocked,
		Summary:  "Caller inventory complete; one hidden bridge still unresolved.",
		NextStep: "Trace bridge registration before deleting old service.",
	})
	require.NoError(t, err)
	require.Equal(t, model.RunStatusBlocked, run.Status)

	resumedRun, resumed, err := svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "codex", AgentTool: model.AgentToolCodex,
	})
	require.NoError(t, err)
	require.True(t, resumed)
	require.Equal(t, run.ID, resumedRun.ID)
	require.Equal(t, model.RunStatusRunning, resumedRun.Status)

	event, err := svc.RecordRunEvent(ctx, service.RecordRunEventInput{
		RunID: resumedRun.ID, AgentName: "codex",
		Kind:    model.RunEventVerificationPassed,
		Summary: "Compile and focused WebView tests passed.",
		Details: map[string]any{"command": "./gradlew test"},
	})
	require.NoError(t, err)
	require.Equal(t, model.RunEventVerificationPassed, event.Kind)

	learningEvent, err := svc.RecordRunEvent(ctx, service.RecordRunEventInput{
		RunID: resumedRun.ID, AgentName: "codex",
		Kind:    model.RunEventLearningDiscovered,
		Summary: "Captured project branch convention",
		Details: map[string]any{
			"kind":     "human-correction",
			"trigger":  "Creating a feature branch for release 7.23.0",
			"guidance": "Name branch 7.23.0_feat/<english-requirement-name>",
			"scope":    "CamScanner feature branches",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, learningEvent.Details["learning_note_id"])
	notes, err := svc.ListLearningNotes(ctx, service.LearningNoteListInput{
		TaskID: task.ID, Status: model.LearningNotePending,
	})
	require.NoError(t, err)
	require.Len(t, notes, 1)
	require.Equal(t, "Name branch 7.23.0_feat/<english-requirement-name>", notes[0].Guidance)

	completed, err := svc.FinishRun(ctx, service.FinishRunInput{
		RunID: resumedRun.ID, AgentName: "codex", Status: model.RunStatusCompleted,
		Summary: "Migration completed and verified.",
	})
	require.NoError(t, err)
	require.NotZero(t, completed.CompletedAt)

	resume, err := svc.GetTaskResumeContext(ctx, task.ID, "codex")
	require.NoError(t, err)
	require.Equal(t, completed.ID, resume.LatestRun.ID)
	require.Equal(t, "Migration completed and verified.", resume.LatestRun.Summary)
	require.NotEmpty(t, resume.RecentEvents)

	metrics, err := svc.GetLearningMetrics(ctx, project.ID)
	require.NoError(t, err)
	require.Equal(t, 1, metrics.RunCount)
	require.Equal(t, 1, metrics.CompletedRunCount)
	require.Equal(t, 1, metrics.BlockedRunCount)
	require.Equal(t, 1, metrics.ResumedRunCount)
	require.Equal(t, 1.0, metrics.RunCompletionRate)
	require.Equal(t, 1.0, metrics.RecoveryRate)
}

func TestAgentRunRequiresLiveTaskOwnership(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "run-owner", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(
		ctx, project.ID, "Owned work", "", model.TaskTypeFeature, 0, true, nil,
	)
	require.NoError(t, err)

	_, _, err = svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "codex", AgentTool: model.AgentToolCodex,
	})
	require.ErrorIs(t, err, store.ErrConflict)

	_, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "alice"})
	require.NoError(t, err)
	_, _, err = svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "bob", AgentTool: model.AgentToolClaudeCode,
	})
	require.ErrorIs(t, err, store.ErrConflict)
}
