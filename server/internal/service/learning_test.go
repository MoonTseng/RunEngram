package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"taskline_server/api/model"
	"taskline_server/internal/service"
)

func TestLearningLoopFreezesContextAndMeasuresHelpfulReuse(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "learning", "")
	require.NoError(t, err)

	source, err := svc.CreateTask(ctx, project.ID, "Migrate URL service", "Move WebView URL methods", model.TaskTypeFeature, 1, true, []string{"webview"})
	require.NoError(t, err)
	capsule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: project.ID, SourceTaskID: source.ID,
		Title:    "WebView URL migration boundary",
		Summary:  "Migrate callers before deleting compatibility service",
		Scope:    "webview service migration",
		Evidence: "All callers enumerated; unit tests and compile passed.",
		Labels:   []string{"webview"}, Fingerprints: []string{"url-service"},
		Producer: "claude-code",
	})
	require.NoError(t, err)
	require.Equal(t, "claude-code", capsule.Producer)

	task, err := svc.CreateTask(ctx, project.ID, "Refactor webview url-service", "Reuse earlier webview migration", model.TaskTypeFeature, 2, true, []string{"webview"})
	require.NoError(t, err)
	task, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)

	first, err := svc.GetOrCreateTaskContext(ctx, task.ID)
	require.NoError(t, err)
	require.Len(t, first.SuggestedCapsules, 1)
	require.Equal(t, capsule.ID, first.SuggestedCapsules[0].ID)

	_, err = svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: project.ID, Title: "Later finding", Summary: "webview url-service later",
		Evidence: "Verified after snapshot.", Labels: []string{"webview"},
	})
	require.NoError(t, err)
	second, err := svc.GetOrCreateTaskContext(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, first.SuggestedCapsules, second.SuggestedCapsules)

	_, err = svc.RecordCapsuleUsage(ctx, service.RecordUsageInput{
		CapsuleID: capsule.ID, TaskID: task.ID,
		Outcome: model.CapsuleOutcomeHelpful, Notes: "avoided repeated caller search",
	})
	require.NoError(t, err)
	metrics, err := svc.GetLearningMetrics(ctx, project.ID)
	require.NoError(t, err)
	require.Equal(t, 2, metrics.CapsuleCount)
	require.Equal(t, 1, metrics.SnapshotTaskCount)
	require.Equal(t, 1, metrics.ReusedTaskCount)
	require.Equal(t, 1, metrics.HelpfulCount)
	require.Equal(t, 1.0, metrics.HelpfulRate)
}

func TestTaskContextRequiresClaimAndUsageCannotCrossProjects(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	left, err := svc.CreateProject(ctx, "left", "")
	require.NoError(t, err)
	right, err := svc.CreateProject(ctx, "right", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(ctx, left.ID, "unclaimed", "", model.TaskTypeFeature, 0, true, nil)
	require.NoError(t, err)
	_, err = svc.GetOrCreateTaskContext(ctx, task.ID)
	require.Error(t, err)

	capsule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: right.ID, Title: "Other project", Summary: "must stay isolated", Evidence: "verified",
	})
	require.NoError(t, err)
	_, err = svc.RecordCapsuleUsage(ctx, service.RecordUsageInput{
		CapsuleID: capsule.ID, TaskID: task.ID, Outcome: model.CapsuleOutcomeUsed,
	})
	require.Error(t, err)

	rightTask, err := svc.CreateTask(ctx, right.ID, "invalidate", "", model.TaskTypeBug, 0, true, nil)
	require.NoError(t, err)
	_, err = svc.RecordCapsuleUsage(ctx, service.RecordUsageInput{
		CapsuleID: capsule.ID, TaskID: rightTask.ID,
		Outcome: model.CapsuleOutcomeStale, Notes: "current code disproves it",
	})
	require.NoError(t, err)
	stale, err := svc.GetCapsule(ctx, capsule.ID)
	require.NoError(t, err)
	require.Equal(t, model.CapsuleStatusStale, stale.Status)
}
