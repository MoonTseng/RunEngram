package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"taskline_server/api/model"
	"taskline_server/internal/store"
)

func memoryImpactFixture(t *testing.T) (*store.Store, *model.Project, *model.Task, *model.ExplorationCapsule) {
	t.Helper()
	ctx := context.Background()
	st := newTestStore(t)
	project, err := st.CreateProject(ctx, "memory-impact", "")
	require.NoError(t, err)
	task, err := st.CreateTask(ctx, project.ID, "Use project build rule", "", model.TaskTypeFeature, 1, model.StateStart)
	require.NoError(t, err)
	capsule := &model.ExplorationCapsule{
		ProjectID: project.ID,
		Title:     "Do not run Gradle",
		Summary:   "Use static inspection unless developer requests a build.",
		Scope:     "CamScanner",
		Evidence:  "Verified in prior task.",
	}
	require.NoError(t, st.CreateCapsule(ctx, capsule))
	return st, project, task, capsule
}

func TestMemoryImpactRecallUpsertPreservesLaterState(t *testing.T) {
	ctx := context.Background()
	st, project, task, capsule := memoryImpactFixture(t)

	impact, err := st.UpsertMemoryImpactRecall(ctx, &model.MemoryImpact{
		ProjectID:       project.ID,
		TaskID:          task.ID,
		CapsuleID:       capsule.ID,
		State:           model.MemoryImpactRecalled,
		RecallSource:    "task-context",
		ContextRevision: "rev-1",
		RecallScore:     0.91,
		RecallReasons:   []string{"label:android", "scope:camscanner"},
	})
	require.NoError(t, err)

	state := model.MemoryImpactApplied
	notes := "Skipped Gradle because project rule forbids local builds."
	applied, err := st.UpdateMemoryImpact(ctx, impact.ID, store.MemoryImpactUpdate{
		State:             &state,
		Notes:             &notes,
		ExpectedUpdatedAt: impact.UpdatedAt,
	})
	require.NoError(t, err)

	again, err := st.UpsertMemoryImpactRecall(ctx, &model.MemoryImpact{
		ProjectID:       project.ID,
		TaskID:          task.ID,
		CapsuleID:       capsule.ID,
		State:           model.MemoryImpactRecalled,
		RecallSource:    "dynamic-recall",
		ContextRevision: "rev-2",
		RecallScore:     0.97,
		RecallReasons:   []string{"query:gradle"},
	})
	require.NoError(t, err)
	require.Equal(t, model.MemoryImpactApplied, again.State)
	require.Equal(t, applied.Notes, again.Notes)
	require.Equal(t, "rev-2", again.ContextRevision)
	require.Equal(t, []string{"query:gradle"}, again.RecallReasons)
}

func TestMemoryImpactListFiltersProjectTaskAndCapsule(t *testing.T) {
	ctx := context.Background()
	st, project, task, capsule := memoryImpactFixture(t)
	_, err := st.UpsertMemoryImpactRecall(ctx, &model.MemoryImpact{
		ProjectID: project.ID, TaskID: task.ID, CapsuleID: capsule.ID,
		State: model.MemoryImpactRecalled,
	})
	require.NoError(t, err)

	for name, filter := range map[string]store.MemoryImpactFilter{
		"project": {ProjectID: project.ID},
		"task":    {TaskID: task.ID},
		"capsule": {CapsuleID: capsule.ID},
		"state":   {ProjectID: project.ID, States: []model.MemoryImpactState{model.MemoryImpactRecalled}},
	} {
		t.Run(name, func(t *testing.T) {
			impacts, err := st.ListMemoryImpacts(ctx, filter)
			require.NoError(t, err)
			require.Len(t, impacts, 1)
			require.Equal(t, task.ID, impacts[0].TaskID)
		})
	}
}

func TestMemoryImpactOptimisticUpdateAndEvidenceRoundTrip(t *testing.T) {
	ctx := context.Background()
	st, project, task, capsule := memoryImpactFixture(t)
	impact, err := st.UpsertMemoryImpactRecall(ctx, &model.MemoryImpact{
		ProjectID: project.ID, TaskID: task.ID, CapsuleID: capsule.ID,
		State: model.MemoryImpactRecalled,
	})
	require.NoError(t, err)

	state := model.MemoryImpactHelpful
	stage := "test"
	actor := "yue_zeng"
	notes := "Static verification completed without a Gradle build."
	evidence := []model.MemoryImpactEvidence{{
		Kind: "task-doc", Ref: "doc:test-report", Summary: "Report confirms no Gradle command ran.",
	}}
	updated, err := st.UpdateMemoryImpact(ctx, impact.ID, store.MemoryImpactUpdate{
		State: &state, Stage: &stage, Actor: &actor, Notes: &notes, Evidence: &evidence,
		ExpectedUpdatedAt: impact.UpdatedAt,
	})
	require.NoError(t, err)
	require.Equal(t, evidence, updated.Evidence)
	require.NotZero(t, updated.ResolvedAt)

	_, err = st.UpdateMemoryImpact(ctx, impact.ID, store.MemoryImpactUpdate{
		State: &state, ExpectedUpdatedAt: impact.UpdatedAt,
	})
	require.ErrorIs(t, err, store.ErrConflict)
}

func TestMemoryImpactHistorySurvivesTaskDeletion(t *testing.T) {
	ctx := context.Background()
	st, project, task, capsule := memoryImpactFixture(t)
	_, err := st.UpsertMemoryImpactRecall(ctx, &model.MemoryImpact{
		ProjectID: project.ID, TaskID: task.ID, CapsuleID: capsule.ID,
		State: model.MemoryImpactRecalled,
	})
	require.NoError(t, err)
	require.NoError(t, st.DeleteTask(ctx, task.ID))

	impacts, err := st.ListMemoryImpacts(ctx, store.MemoryImpactFilter{CapsuleID: capsule.ID})
	require.NoError(t, err)
	require.Len(t, impacts, 1)
	require.Equal(t, task.ID, impacts[0].TaskID)
}

func TestMemoryImpactMetricsCountDistinctTasks(t *testing.T) {
	ctx := context.Background()
	st, project, task, capsule := memoryImpactFixture(t)
	first, err := st.UpsertMemoryImpactRecall(ctx, &model.MemoryImpact{
		ProjectID: project.ID, TaskID: task.ID, CapsuleID: capsule.ID,
		State: model.MemoryImpactRecalled,
	})
	require.NoError(t, err)

	capsule2 := &model.ExplorationCapsule{
		ProjectID: project.ID, Title: "Second rule", Summary: "Another rule", Evidence: "Verified.",
	}
	require.NoError(t, st.CreateCapsule(ctx, capsule2))
	second, err := st.UpsertMemoryImpactRecall(ctx, &model.MemoryImpact{
		ProjectID: project.ID, TaskID: task.ID, CapsuleID: capsule2.ID,
		State: model.MemoryImpactRecalled,
	})
	require.NoError(t, err)

	applied := model.MemoryImpactApplied
	_, err = st.UpdateMemoryImpact(ctx, first.ID, store.MemoryImpactUpdate{
		State: &applied, ExpectedUpdatedAt: first.UpdatedAt,
	})
	require.NoError(t, err)
	helpful := model.MemoryImpactHelpful
	_, err = st.UpdateMemoryImpact(ctx, second.ID, store.MemoryImpactUpdate{
		State: &helpful, ExpectedUpdatedAt: second.UpdatedAt,
	})
	require.NoError(t, err)

	metrics, err := st.GetMemoryImpactMetrics(ctx, project.ID)
	require.NoError(t, err)
	require.Equal(t, 1, metrics.RecalledTaskCount)
	require.Equal(t, 2, metrics.RecalledMemoryCount)
	require.Equal(t, 1, metrics.AppliedTaskCount)
	require.Equal(t, 1, metrics.HelpfulTaskCount)
}

func TestMarkTaskMemoryImpactsUnconfirmedOnlyTouchesUnresolved(t *testing.T) {
	ctx := context.Background()
	st, project, task, capsule := memoryImpactFixture(t)
	impact, err := st.UpsertMemoryImpactRecall(ctx, &model.MemoryImpact{
		ProjectID: project.ID, TaskID: task.ID, CapsuleID: capsule.ID,
		State: model.MemoryImpactRecalled,
	})
	require.NoError(t, err)

	require.NoError(t, st.MarkTaskMemoryImpactsUnconfirmed(ctx, task.ID, "system"))
	got, err := st.GetMemoryImpact(ctx, impact.ID)
	require.NoError(t, err)
	require.Equal(t, model.MemoryImpactUnconfirmed, got.State)

	helpful := model.MemoryImpactHelpful
	got, err = st.UpdateMemoryImpact(ctx, got.ID, store.MemoryImpactUpdate{
		State: &helpful, ExpectedUpdatedAt: got.UpdatedAt,
	})
	require.NoError(t, err)
	require.NoError(t, st.MarkTaskMemoryImpactsUnconfirmed(ctx, task.ID, "system"))
	got, err = st.GetMemoryImpact(ctx, got.ID)
	require.NoError(t, err)
	require.Equal(t, model.MemoryImpactHelpful, got.State)
}
