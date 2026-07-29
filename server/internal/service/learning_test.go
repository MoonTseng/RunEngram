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
		Guidance: "Use project/requirement-import",
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

	first, err := svc.PromoteLearningNote(ctx, note.ID, "other-agent", "notion-to-prd produced the PRD; tests passed")
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

func TestPendingLearningNoteCanBeCorrectedBeforePromotion(t *testing.T) {
	ctx := service.WithActor(context.Background(), "codex")
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "editable-learning", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(
		ctx, project.ID, "Follow project conventions", "", model.TaskTypeFeature, 1, true, nil,
	)
	require.NoError(t, err)
	_, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)
	note, err := svc.CaptureLearningNote(ctx, service.CaptureLearningNoteInput{
		ProjectID: project.ID, SourceTaskID: task.ID, AgentName: "codex",
		Kind: model.LearningNoteHumanCorrection, Trigger: "Create feature branch",
		Guidance: "Use version prefix", Scope: "All feature branches", Producer: "codex",
	})
	require.NoError(t, err)

	updated, err := svc.UpdateLearningNote(ctx, note.ID, "codex", service.UpdateLearningNoteInput{
		Trigger:  "Creating a feature branch for release 7.23.0",
		Guidance: "Name branch 7.23.0_feat/<english-requirement-name>",
		Scope:    "CamScanner feature branches",
	})
	require.NoError(t, err)
	require.Equal(t, "Name branch 7.23.0_feat/<english-requirement-name>", updated.Guidance)
	require.Equal(t, "CamScanner feature branches", updated.Scope)

	_, err = svc.UpdateLearningNote(ctx, note.ID, "reviewer", service.UpdateLearningNoteInput{
		Trigger: updated.Trigger, Guidance: updated.Guidance, Scope: updated.Scope,
	})
	require.NoError(t, err)
	_, err = svc.PromoteLearningNote(ctx, note.ID, "codex", "Branch created and verified")
	require.NoError(t, err)
	_, err = svc.UpdateLearningNote(ctx, note.ID, "codex", service.UpdateLearningNoteInput{
		Trigger: "late edit", Guidance: "late edit",
	})
	require.ErrorIs(t, err, store.ErrConflict)

	events, err := svc.ListTaskEvents(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, "learning_note_promoted", events[0].Action)
	require.Equal(t, "learning_note_updated", events[1].Action)
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
	first, err := svc.RejectLearningNote(ctx, note.ID, "reviewer", "not reusable")
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

func TestContextSeparatesProjectRulesAndUsesAdaptiveExperienceBudget(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "layered-memory", "")
	require.NoError(t, err)

	rule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID:   project.ID,
		Title:       "Branch naming policy",
		Trigger:     "Before creating a feature branch",
		Summary:     "Use release_feat/english-name for feature branches",
		Scope:       "All feature work",
		Evidence:    "Repository branch policy verified by maintainer.",
		MemoryClass: model.MemoryClassProjectRule,
	})
	require.NoError(t, err)

	for i := 0; i < 12; i++ {
		_, err = svc.CreateCapsule(ctx, service.CreateCapsuleInput{
			ProjectID:   project.ID,
			Title:       "WebView migration experience",
			Trigger:     "When migrating WebView service callers",
			Summary:     "Enumerate WebView callers before deleting compatibility service",
			Scope:       "WebView module migration",
			Evidence:    "Caller inventory and module compilation passed.",
			Labels:      []string{"webview"},
			MemoryClass: model.MemoryClassExperience,
		})
		require.NoError(t, err)
	}

	task, err := svc.CreateTask(
		ctx,
		project.ID,
		"Migrate WebView URL service",
		"Move callers across WebView modules",
		model.TaskTypeFeature,
		1,
		true,
		[]string{"webview"},
	)
	require.NoError(t, err)
	_, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)

	snapshot, err := svc.GetOrCreateTaskContext(ctx, task.ID)
	require.NoError(t, err)
	require.Len(t, snapshot.ProjectRules, 1)
	require.Equal(t, rule.ID, snapshot.ProjectRules[0].ID)
	require.Greater(t, len(snapshot.SuggestedCapsules), 5)
	require.LessOrEqual(t, len(snapshot.SuggestedCapsules), 20)
}

func TestDynamicRecallFindsExperienceAfterTaskStart(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "dynamic-recall", "")
	require.NoError(t, err)
	capsule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID:   project.ID,
		Title:       "Gradle multi-module recovery",
		Trigger:     "Gradle daemon fails during multi-module compilation",
		Summary:     "Compile modules serially with daemon and parallelism disabled",
		Scope:       "Android multi-module verification",
		Evidence:    "Three affected modules compiled successfully.",
		MemoryClass: model.MemoryClassExperience,
	})
	require.NoError(t, err)

	task, err := svc.CreateTask(ctx, project.ID, "Refactor services", "", model.TaskTypeFeature, 1, true, nil)
	require.NoError(t, err)
	_, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)

	snapshot, err := svc.GetOrCreateTaskContext(ctx, task.ID)
	require.NoError(t, err)
	require.Empty(t, snapshot.SuggestedCapsules)

	recall, err := svc.RecallTaskMemory(ctx, task.ID, "codex", "Gradle daemon multi-module compilation failed")
	require.NoError(t, err)
	require.Len(t, recall.SuggestedCapsules, 1)
	require.Equal(t, capsule.ID, recall.SuggestedCapsules[0].ID)
}

func TestVerifiedExperienceBecomesTrustedThroughHelpfulReuse(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "memory-confidence", "")
	require.NoError(t, err)
	capsule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID:   project.ID,
		Title:       "Verified recovery",
		Trigger:     "Known build failure appears",
		Summary:     "Use verified recovery command",
		Evidence:    "Recovery command passed on source task.",
		MemoryClass: model.MemoryClassExperience,
	})
	require.NoError(t, err)
	require.Equal(t, model.MemoryValidationVerified, capsule.Validation)
	initialConfidence := capsule.Confidence

	for _, title := range []string{"First reuse", "Second reuse"} {
		task, createErr := svc.CreateTask(ctx, project.ID, title, "", model.TaskTypeBug, 1, true, nil)
		require.NoError(t, createErr)
		_, createErr = svc.RecordCapsuleUsage(ctx, service.RecordUsageInput{
			CapsuleID: capsule.ID,
			TaskID:    task.ID,
			Outcome:   model.CapsuleOutcomeHelpful,
			Notes:     "Current code and verification confirmed guidance.",
		})
		require.NoError(t, createErr)
	}

	trusted, err := svc.GetCapsule(ctx, capsule.ID)
	require.NoError(t, err)
	require.Equal(t, model.MemoryValidationTrusted, trusted.Validation)
	require.Greater(t, trusted.Confidence, initialConfidence)
}

func TestDisputedExperienceIsExcludedFromAutomaticRecall(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "disputed-memory", "")
	require.NoError(t, err)
	capsule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: project.ID, Title: "Legacy Gradle recovery",
		Trigger: "Gradle daemon failed", Summary: "Delete every Gradle cache",
		Scope: "Android builds", Evidence: "Worked on one old checkout",
	})
	require.NoError(t, err)

	for _, title := range []string{"First failed reuse", "Second failed reuse"} {
		task, createErr := svc.CreateTask(ctx, project.ID, title, "Gradle daemon failed", model.TaskTypeBug, 1, true, nil)
		require.NoError(t, createErr)
		_, createErr = svc.RecordCapsuleUsage(ctx, service.RecordUsageInput{
			CapsuleID: capsule.ID, TaskID: task.ID, Outcome: model.CapsuleOutcomeRejected,
			Notes: "Current build disproved this route",
		})
		require.NoError(t, createErr)
	}

	target, err := svc.CreateTask(ctx, project.ID, "Another Gradle failure", "Gradle daemon failed", model.TaskTypeBug, 1, true, nil)
	require.NoError(t, err)
	_, err = svc.ClaimTask(ctx, target.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)
	snapshot, err := svc.GetOrCreateTaskContext(ctx, target.ID)
	require.NoError(t, err)
	require.Empty(t, snapshot.SuggestedCapsules)

	refreshed, err := svc.GetCapsule(ctx, capsule.ID)
	require.NoError(t, err)
	require.Equal(t, model.MemoryValidationDisputed, refreshed.Validation)
}

func TestReviewerCanPromoteExpiredCandidateAsProjectRule(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "platform-review", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(ctx, project.ID, "Capture branch policy", "", model.TaskTypeFeature, 1, true, nil)
	require.NoError(t, err)
	_, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex", Lease: time.Millisecond})
	require.NoError(t, err)
	note, err := svc.CaptureLearningNote(ctx, service.CaptureLearningNoteInput{
		ProjectID:    project.ID,
		SourceTaskID: task.ID,
		AgentName:    "codex",
		Kind:         model.LearningNoteHumanCorrection,
		Trigger:      "Before creating a feature branch",
		Guidance:     "Use release_feat/english-name",
		Scope:        "All feature work",
	})
	require.NoError(t, err)
	time.Sleep(5 * time.Millisecond)

	promoted, err := svc.PromoteLearningNote(
		ctx,
		note.ID,
		"maintainer",
		"Maintainer checked repository branch policy.",
		model.MemoryClassProjectRule,
	)
	require.NoError(t, err)
	capsule, err := svc.GetCapsule(ctx, promoted.CapsuleID)
	require.NoError(t, err)
	require.Equal(t, model.MemoryClassProjectRule, capsule.MemoryClass)
	require.Equal(t, note.Trigger, capsule.Trigger)
}
