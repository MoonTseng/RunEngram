package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"taskline_server/api/model"
	"taskline_server/internal/service"
	"taskline_server/internal/store"
)

func newMemoryImpactSvc(t *testing.T) (*service.Service, *store.Store) {
	t.Helper()
	st, err := store.New(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	return service.New(st), st
}

func memoryImpactServiceFixture(
	t *testing.T,
) (*service.Service, *store.Store, *model.Project, *model.Task, *model.ExplorationCapsule) {
	t.Helper()
	ctx := context.Background()
	svc, st := newMemoryImpactSvc(t)
	project, err := svc.CreateProject(ctx, "memory-impact-service", "")
	require.NoError(t, err)
	capsule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID:   project.ID,
		MemoryClass: model.MemoryClassProjectRule,
		Trigger:     "Every CamScanner task",
		Title:       "Do not run Gradle",
		Summary:     "Use static inspection unless developer requests a build.",
		Scope:       "CamScanner",
		Evidence:    "Verified project policy.",
	})
	require.NoError(t, err)
	task, err := svc.CreateTask(
		ctx, project.ID, "Inspect password flow", "Do not build", model.TaskTypeFeature, 1, true, nil,
	)
	require.NoError(t, err)
	task, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)
	return svc, st, project, task, capsule
}

func TestTaskContextCreatesRecallReceipts(t *testing.T) {
	ctx := context.Background()
	svc, st, _, task, capsule := memoryImpactServiceFixture(t)

	snapshot, err := svc.GetOrCreateTaskContext(ctx, task.ID)
	require.NoError(t, err)
	require.Len(t, snapshot.ProjectRules, 1)

	impacts, err := st.ListMemoryImpacts(ctx, store.MemoryImpactFilter{TaskID: task.ID})
	require.NoError(t, err)
	require.Len(t, impacts, 1)
	require.Equal(t, capsule.ID, impacts[0].CapsuleID)
	require.Equal(t, model.MemoryImpactRecalled, impacts[0].State)
	require.Equal(t, "task-context", impacts[0].RecallSource)
	require.Equal(t, snapshot.ContextRevision, impacts[0].ContextRevision)
}

func TestDynamicRecallUpsertsWithoutResettingAppliedReceipt(t *testing.T) {
	ctx := context.Background()
	svc, st, _, task, _ := memoryImpactServiceFixture(t)
	_, err := svc.GetOrCreateTaskContext(ctx, task.ID)
	require.NoError(t, err)
	impacts, err := st.ListMemoryImpacts(ctx, store.MemoryImpactFilter{TaskID: task.ID})
	require.NoError(t, err)

	applied, err := svc.RecordMemoryImpact(ctx, service.RecordMemoryImpactInput{
		ImpactID:          impacts[0].ID,
		State:             model.MemoryImpactApplied,
		Stage:             "dev",
		Notes:             "Skipped Gradle and used static inspection.",
		Actor:             "codex",
		AgentName:         "codex",
		ExpectedUpdatedAt: impacts[0].UpdatedAt,
	})
	require.NoError(t, err)

	_, err = svc.RecallTaskMemory(ctx, task.ID, "codex", "Should I run Gradle?")
	require.NoError(t, err)
	got, err := st.GetMemoryImpact(ctx, impacts[0].ID)
	require.NoError(t, err)
	require.Equal(t, model.MemoryImpactApplied, got.State)
	require.Equal(t, applied.Notes, got.Notes)
	require.Equal(t, "dynamic-recall", got.RecallSource)
}

func TestRecordMemoryImpactRequiresNotesAndTerminalEvidence(t *testing.T) {
	ctx := context.Background()
	svc, st, _, task, _ := memoryImpactServiceFixture(t)
	_, err := svc.GetOrCreateTaskContext(ctx, task.ID)
	require.NoError(t, err)
	impacts, err := st.ListMemoryImpacts(ctx, store.MemoryImpactFilter{TaskID: task.ID})
	require.NoError(t, err)

	_, err = svc.RecordMemoryImpact(ctx, service.RecordMemoryImpactInput{
		ImpactID: impacts[0].ID, State: model.MemoryImpactApplied,
		AgentName: "codex", ExpectedUpdatedAt: impacts[0].UpdatedAt,
	})
	require.ErrorContains(t, err, "notes required")

	_, err = svc.RecordMemoryImpact(ctx, service.RecordMemoryImpactInput{
		ImpactID: impacts[0].ID, State: model.MemoryImpactHelpful,
		Notes: "Rule changed verification strategy.", Actor: "codex", AgentName: "codex",
		ExpectedUpdatedAt: impacts[0].UpdatedAt,
	})
	require.ErrorContains(t, err, "evidence required")
}

func TestDeveloperCanCorrectTerminalImpactButAgentCannot(t *testing.T) {
	ctx := context.Background()
	svc, st, _, task, _ := memoryImpactServiceFixture(t)
	_, err := svc.GetOrCreateTaskContext(ctx, task.ID)
	require.NoError(t, err)
	impacts, err := st.ListMemoryImpacts(ctx, store.MemoryImpactFilter{TaskID: task.ID})
	require.NoError(t, err)
	evidence := []model.MemoryImpactEvidence{{
		Kind: "task-doc", Ref: "doc:test-report", Summary: "Report confirms no build.",
	}}
	helpful, err := svc.RecordMemoryImpact(ctx, service.RecordMemoryImpactInput{
		ImpactID: impacts[0].ID, State: model.MemoryImpactHelpful,
		Notes: "Rule prevented an unsupported build.", Evidence: evidence,
		Actor: "codex", AgentName: "codex", ExpectedUpdatedAt: impacts[0].UpdatedAt,
	})
	require.NoError(t, err)

	_, err = svc.RecordMemoryImpact(ctx, service.RecordMemoryImpactInput{
		ImpactID: helpful.ID, State: model.MemoryImpactRejected,
		Notes: "Later result contradicted it.", Evidence: evidence,
		Actor: "codex", AgentName: "codex", ExpectedUpdatedAt: helpful.UpdatedAt,
	})
	require.ErrorContains(t, err, "agent cannot overwrite")

	correctedEvidence := []model.MemoryImpactEvidence{{
		Kind: "observation", Ref: "manual-review", Summary: "Developer confirmed rule did not apply.",
	}}
	corrected, err := svc.RecordMemoryImpact(ctx, service.RecordMemoryImpactInput{
		ImpactID: helpful.ID, State: model.MemoryImpactRejected,
		Notes: "Rule did not apply to this repository.", Evidence: correctedEvidence,
		Actor: "yue_zeng", ExpectedUpdatedAt: helpful.UpdatedAt,
	})
	require.NoError(t, err)
	require.Equal(t, model.MemoryImpactRejected, corrected.State)
}

func TestRecordUsageMirrorsHelpfulReceiptAndCapsuleUsage(t *testing.T) {
	ctx := context.Background()
	svc, st, project, task, capsule := memoryImpactServiceFixture(t)
	_, err := svc.GetOrCreateTaskContext(ctx, task.ID)
	require.NoError(t, err)

	_, err = svc.RecordCapsuleUsage(ctx, service.RecordUsageInput{
		CapsuleID: capsule.ID,
		TaskID:    task.ID,
		Outcome:   model.CapsuleOutcomeHelpful,
		Notes:     "Rule avoided repeated build failure.",
		Evidence: []model.MemoryImpactEvidence{{
			Kind: "task-doc", Ref: "doc:test-report", Summary: "Static verification passed.",
		}},
		Actor: "codex",
	})
	require.NoError(t, err)

	impacts, err := st.ListMemoryImpacts(ctx, store.MemoryImpactFilter{TaskID: task.ID})
	require.NoError(t, err)
	require.Len(t, impacts, 1)
	require.Equal(t, model.MemoryImpactHelpful, impacts[0].State)
	metrics, err := svc.GetLearningMetrics(ctx, project.ID)
	require.NoError(t, err)
	require.Equal(t, 1, metrics.HelpfulCount)
}

func TestDoneMarksUnresolvedImpactsUnconfirmedWithoutBlockingTask(t *testing.T) {
	ctx := context.Background()
	svc, st, _, task, _ := memoryImpactServiceFixture(t)
	_, err := svc.GetOrCreateTaskContext(ctx, task.ID)
	require.NoError(t, err)

	done := model.StateDone
	task, err = svc.UpdateTask(ctx, task.ID, store.TaskUpdate{
		State: &done, Owner: "codex",
	})
	require.NoError(t, err)
	require.Equal(t, model.StateDone, task.State)
	impacts, err := st.ListMemoryImpacts(ctx, store.MemoryImpactFilter{TaskID: task.ID})
	require.NoError(t, err)
	require.Equal(t, model.MemoryImpactUnconfirmed, impacts[0].State)
}

func TestReconcileSnapshotsCreatesOnlyMissingRecallReceipts(t *testing.T) {
	ctx := context.Background()
	svc, st := newMemoryImpactSvc(t)
	project, err := svc.CreateProject(ctx, "legacy-snapshot", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(ctx, project.ID, "Legacy task", "", model.TaskTypeFeature, 1, true, nil)
	require.NoError(t, err)
	capsule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: project.ID, Title: "Legacy memory", Summary: "Old context memory",
		Evidence: "Verified.",
	})
	require.NoError(t, err)
	_, err = st.CreateContextSnapshot(ctx, &model.ContextSnapshot{
		TaskID: task.ID, ProjectID: project.ID, Task: *task,
		SuggestedCapsules: []model.ExplorationCapsule{*capsule},
		ContextRevision:   "legacy-revision",
	})
	require.NoError(t, err)

	require.NoError(t, svc.ReconcileMemoryImpacts(ctx, project.ID))
	require.NoError(t, svc.ReconcileMemoryImpacts(ctx, project.ID))
	impacts, err := st.ListMemoryImpacts(ctx, store.MemoryImpactFilter{ProjectID: project.ID})
	require.NoError(t, err)
	require.Len(t, impacts, 1)
	require.Equal(t, "snapshot-backfill", impacts[0].RecallSource)
	require.Equal(t, model.MemoryImpactRecalled, impacts[0].State)
}
