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

func TestMemoryRelationsSupersedeOldKnowledgeAndExplainRecall(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "memory-graph", "")
	require.NoError(t, err)

	oldCapsule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: project.ID, Title: "Gradle module verification",
		Trigger: "multi module build", Summary: "Compile every module in parallel",
		Scope: "Android modules", Evidence: "Legacy task output",
		MemoryClass: model.MemoryClassExperience,
	})
	require.NoError(t, err)
	newCapsule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: project.ID, Title: "Gradle module verification",
		Trigger: "multi module build", Summary: "Compile modules serially with parallelism disabled",
		Scope: "Android modules", Evidence: "Verified on three modules",
		MemoryClass: model.MemoryClassExperience,
	})
	require.NoError(t, err)

	relation, err := svc.CreateMemoryRelation(ctx, newCapsule.ID, service.CreateMemoryRelationInput{
		Type:       model.MemoryRelationSupersedes,
		TargetKind: model.MemoryRelationTargetCapsule,
		TargetRef:  oldCapsule.ID,
		Note:       "Parallel compilation proved flaky",
	})
	require.NoError(t, err)
	require.Equal(t, newCapsule.ID, relation.SourceCapsuleID)

	oldCapsule, err = svc.GetCapsule(ctx, oldCapsule.ID)
	require.NoError(t, err)
	require.Equal(t, model.CapsuleStatusStale, oldCapsule.Status)
	require.Len(t, oldCapsule.Relations, 1)
	require.Equal(t, model.MemoryRelationIncoming, oldCapsule.Relations[0].Direction)

	_, err = svc.CreateMemoryRelation(ctx, newCapsule.ID, service.CreateMemoryRelationInput{
		Type:       model.MemoryRelationAppliesTo,
		TargetKind: model.MemoryRelationTargetScope,
		TargetRef:  "billing-module",
		Note:       "Validated module boundary",
	})
	require.NoError(t, err)
	conflictingCapsule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: project.ID, Title: "Legacy release naming",
		Trigger: "release branch", Summary: "Use a legacy release branch prefix",
		Scope: "iOS release", Evidence: "Old release notes",
		MemoryClass: model.MemoryClassExperience,
	})
	require.NoError(t, err)
	_, err = svc.CreateMemoryRelation(ctx, conflictingCapsule.ID, service.CreateMemoryRelationInput{
		Type:       model.MemoryRelationConflictsWith,
		TargetKind: model.MemoryRelationTargetCapsule,
		TargetRef:  newCapsule.ID,
		Note:       "Needs project-owner resolution",
	})
	require.NoError(t, err)

	task, err := svc.CreateTask(ctx, project.ID, "Repair billing build", "Gradle multi module compilation failed", model.TaskTypeBug, 1, true, nil)
	require.NoError(t, err)
	_, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)

	recall, err := svc.RecallTaskMemory(ctx, task.ID, "codex", "billing-module Gradle verification")
	require.NoError(t, err)
	require.NotEmpty(t, recall.ContextRevision)
	require.Len(t, recall.SuggestedCapsules, 1)
	require.Equal(t, newCapsule.ID, recall.SuggestedCapsules[0].ID)
	require.Len(t, recall.Explanations, 1)
	require.Equal(t, newCapsule.ID, recall.Explanations[0].CapsuleID)
	require.NotEmpty(t, recall.Explanations[0].Reasons)
	require.Contains(t, recall.Explanations[0].Reasons, model.MemoryRecallReason{
		Code: "applies-to", Value: "billing-module",
	})
	require.Contains(t, recall.Explanations[0].Warnings, "conflicts-with:"+conflictingCapsule.ID)
}

func TestMemoryRelationRejectsInvalidGraphEdges(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "memory-graph-guards", "")
	require.NoError(t, err)
	otherProject, err := svc.CreateProject(ctx, "memory-graph-other", "")
	require.NoError(t, err)

	first, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: project.ID, Title: "First", Summary: "First rule", Evidence: "verified",
	})
	require.NoError(t, err)
	second, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: project.ID, Title: "Second", Summary: "Second rule", Evidence: "verified",
	})
	require.NoError(t, err)
	foreign, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: otherProject.ID, Title: "Foreign", Summary: "Foreign rule", Evidence: "verified",
	})
	require.NoError(t, err)

	_, err = svc.CreateMemoryRelation(ctx, first.ID, service.CreateMemoryRelationInput{
		Type: model.MemoryRelationConflictsWith, TargetKind: model.MemoryRelationTargetCapsule, TargetRef: first.ID,
	})
	require.Error(t, err)

	_, err = svc.CreateMemoryRelation(ctx, first.ID, service.CreateMemoryRelationInput{
		Type: model.MemoryRelationSupersedes, TargetKind: model.MemoryRelationTargetCapsule, TargetRef: foreign.ID,
	})
	require.Error(t, err)

	_, err = svc.CreateMemoryRelation(ctx, first.ID, service.CreateMemoryRelationInput{
		Type: model.MemoryRelationSupersedes, TargetKind: model.MemoryRelationTargetCapsule, TargetRef: second.ID,
	})
	require.NoError(t, err)
	_, err = svc.CreateMemoryRelation(ctx, second.ID, service.CreateMemoryRelationInput{
		Type: model.MemoryRelationSupersedes, TargetKind: model.MemoryRelationTargetCapsule, TargetRef: first.ID,
	})
	require.Error(t, err)
}

func TestCapsuleUpdateUsesOptimisticConcurrency(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "memory-cas", "")
	require.NoError(t, err)
	capsule, err := svc.CreateCapsule(ctx, service.CreateCapsuleInput{
		ProjectID: project.ID, Title: "Original", Summary: "Original summary", Evidence: "verified",
	})
	require.NoError(t, err)

	firstVersion := capsule.UpdatedAt
	title := "Corrected"
	capsule, err = svc.UpdateCapsule(ctx, capsule.ID, service.UpdateCapsuleInput{
		Title: &title, ExpectedUpdatedAt: &firstVersion,
	})
	require.NoError(t, err)
	require.Equal(t, title, capsule.Title)

	staleTitle := "Stale overwrite"
	_, err = svc.UpdateCapsule(ctx, capsule.ID, service.UpdateCapsuleInput{
		Title: &staleTitle, ExpectedUpdatedAt: &firstVersion,
	})
	require.ErrorIs(t, err, store.ErrConflict)
}
