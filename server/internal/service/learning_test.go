package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"taskline_server/api/model"
	"taskline_server/internal/service"
	"taskline_server/internal/store"
)

func TestLearningNoteRequiresClaimAndPromotesOnce(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "auto-learning", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(
		ctx,
		project.ID,
		"Read Notion PRD",
		"Notion requirement",
		model.TaskTypeFeature,
		1,
		true,
		[]string{"notion"},
	)
	require.NoError(t, err)

	input := service.CaptureLearningNoteInput{
		ProjectID: project.ID, SourceTaskID: task.ID, AgentName: "codex",
		Kind:     model.LearningNoteHumanCorrection,
		Trigger:  "Notion link was not readable",
		Guidance: "Use one-flow/notion-to-prd",
		Scope:    "Notion requirements",
		Labels:   []string{"notion"}, Fingerprints: []string{"notion-to-prd"},
		Producer: "codex",
	}
	_, err = svc.CaptureLearningNote(ctx, input)
	require.ErrorIs(t, err, store.ErrConflict)

	task, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)
	note, err := svc.CaptureLearningNote(ctx, input)
	require.NoError(t, err)
	require.Equal(t, model.LearningNotePending, note.Status)

	_, err = svc.PromoteLearningNote(ctx, note.ID, "other-agent", "verified")
	require.ErrorIs(t, err, store.ErrConflict)
	first, err := svc.PromoteLearningNote(ctx, note.ID, "codex", "notion-to-prd produced the PRD; tests passed")
	require.NoError(t, err)
	second, err := svc.PromoteLearningNote(ctx, note.ID, "codex", "same retry")
	require.NoError(t, err)
	require.Equal(t, first.CapsuleID, second.CapsuleID)

	capsules, err := svc.ListCapsules(ctx, service.CapsuleListInput{
		ProjectID: project.ID, Status: model.CapsuleStatusActive,
	})
	require.NoError(t, err)
	require.Len(t, capsules, 1)
	metrics, err := svc.GetLearningMetrics(ctx, project.ID)
	require.NoError(t, err)
	require.Equal(t, 1, metrics.LearningNoteCount)
	require.Equal(t, 0, metrics.PendingNoteCount)
	require.Equal(t, 1, metrics.PromotedNoteCount)
	require.Equal(t, 0, metrics.RejectedNoteCount)
	require.Equal(t, 1.0, metrics.PromotionRate)

	_, err = svc.RejectLearningNote(ctx, note.ID, "codex", "already promoted")
	require.ErrorIs(t, err, store.ErrConflict)
	events, err := svc.ListTaskEvents(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, "learning_note_promoted", events[0].Action)
	require.Equal(t, "learning_note_captured", events[1].Action)
}

func TestLearningNoteRejectionAndExpiredClaimGuards(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "learning-guards", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(ctx, project.ID, "Try fallback", "", model.TaskTypeBug, 1, true, nil)
	require.NoError(t, err)
	_, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)
	note, err := svc.CaptureLearningNote(ctx, service.CaptureLearningNoteInput{
		ProjectID: project.ID, SourceTaskID: task.ID, AgentName: "codex",
		Kind: model.LearningNoteAgentRecovery, Trigger: "Primary path failed",
		Guidance: "Use verified fallback", Producer: "codex",
	})
	require.NoError(t, err)

	_, err = svc.RejectLearningNote(ctx, note.ID, "codex", "")
	require.Error(t, err)
	_, err = svc.RejectLearningNote(ctx, note.ID, "other-agent", "not reusable")
	require.ErrorIs(t, err, store.ErrConflict)
	first, err := svc.RejectLearningNote(ctx, note.ID, "codex", "not reusable")
	require.NoError(t, err)
	second, err := svc.RejectLearningNote(ctx, note.ID, "codex", "retry")
	require.NoError(t, err)
	require.Equal(t, first.RejectionReason, second.RejectionReason)
	_, err = svc.PromoteLearningNote(ctx, note.ID, "codex", "late evidence")
	require.ErrorIs(t, err, store.ErrConflict)
	metrics, err := svc.GetLearningMetrics(ctx, project.ID)
	require.NoError(t, err)
	require.Equal(t, 1, metrics.LearningNoteCount)
	require.Equal(t, 1, metrics.RejectedNoteCount)
	require.Equal(t, 0.0, metrics.PromotionRate)

	expired, err := svc.CreateTask(ctx, project.ID, "Expired claim", "", model.TaskTypeBug, 1, true, nil)
	require.NoError(t, err)
	_, err = svc.ClaimTask(ctx, expired.ID, service.ClaimOptions{Owner: "codex", Lease: time.Millisecond})
	require.NoError(t, err)
	time.Sleep(5 * time.Millisecond)
	_, err = svc.CaptureLearningNote(ctx, service.CaptureLearningNoteInput{
		ProjectID: project.ID, SourceTaskID: expired.ID, AgentName: "codex",
		Kind: model.LearningNoteAgentRecovery, Trigger: "Expired", Guidance: "Should fail",
	})
	require.ErrorIs(t, err, store.ErrConflict)
}

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
