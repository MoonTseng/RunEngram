package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"taskline_server/api/model"
	"taskline_server/internal/store"
)

type CreateCapsuleInput struct {
	ProjectID    string
	SourceTaskID string
	MemoryClass  model.MemoryClass
	Trigger      string
	Title        string
	Summary      string
	Scope        string
	Evidence     string
	Labels       []string
	Fingerprints []string
	Producer     string
}

type UpdateCapsuleInput struct {
	MemoryClass       *model.MemoryClass
	Trigger           *string
	Title             *string
	Summary           *string
	Scope             *string
	Evidence          *string
	Labels            *[]string
	Fingerprints      *[]string
	Status            *model.CapsuleStatus
	ExpectedUpdatedAt *int64
}

type CreateMemoryRelationInput struct {
	Type       model.MemoryRelationType
	TargetKind model.MemoryRelationTargetKind
	TargetRef  string
	Note       string
}

type CapsuleListInput struct {
	ProjectID   string
	Query       string
	Status      model.CapsuleStatus
	MemoryClass model.MemoryClass
	Limit       int
}

type RecordUsageInput struct {
	CapsuleID string
	TaskID    string
	Outcome   model.CapsuleOutcome
	Notes     string
	Stage     string
	Evidence  []model.MemoryImpactEvidence
	Actor     string
	AgentName string
}

type CaptureLearningNoteInput struct {
	ProjectID    string
	SourceTaskID string
	AgentName    string
	Kind         model.LearningNoteKind
	Trigger      string
	Guidance     string
	Scope        string
	Labels       []string
	Fingerprints []string
	Producer     string
}

type LearningNoteListInput struct {
	ProjectID string
	TaskID    string
	Status    model.LearningNoteStatus
	Limit     int
}

type UpdateLearningNoteInput struct {
	Trigger  string
	Guidance string
	Scope    string
}

const (
	maxLearningNoteTriggerRunes   = 2_000
	maxLearningNoteGuidanceRunes  = 8_000
	maxLearningNoteScopeRunes     = 2_000
	maxLearningNoteEvidenceRunes  = 32_000
	maxLearningNoteRejectionRunes = 2_000
	projectRuleBudgetRunes        = 8_000
	experienceRecallBudgetRunes   = 12_000
	maxProjectRules               = 32
	maxRecalledExperiences        = 20
)

func (s *Service) CaptureLearningNote(
	ctx context.Context,
	input CaptureLearningNoteInput,
) (*model.LearningNote, error) {
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.SourceTaskID = strings.TrimSpace(input.SourceTaskID)
	input.AgentName = strings.TrimSpace(input.AgentName)
	input.Trigger = strings.TrimSpace(input.Trigger)
	input.Guidance = strings.TrimSpace(input.Guidance)
	input.Scope = strings.TrimSpace(input.Scope)
	input.Producer = strings.TrimSpace(input.Producer)
	if !input.Kind.Valid() {
		return nil, fmt.Errorf("invalid learning note kind %q", input.Kind)
	}
	if err := validateLearningText("trigger", input.Trigger, true, maxLearningNoteTriggerRunes); err != nil {
		return nil, err
	}
	if err := validateLearningText("guidance", input.Guidance, true, maxLearningNoteGuidanceRunes); err != nil {
		return nil, err
	}
	if err := validateLearningText("scope", input.Scope, false, maxLearningNoteScopeRunes); err != nil {
		return nil, err
	}
	if input.Producer == "" {
		input.Producer = "codex"
	}
	project, err := s.ResolveProject(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}
	task, err := s.st.GetTask(ctx, input.SourceTaskID)
	if err != nil {
		return nil, err
	}
	if task.ProjectID != project.ID {
		return nil, fmt.Errorf("%w: source task belongs to another project", store.ErrConflict)
	}
	if err := requireLiveOwner(task, input.AgentName); err != nil {
		return nil, err
	}
	note := &model.LearningNote{
		ProjectID: project.ID, SourceTaskID: task.ID, Kind: input.Kind,
		Trigger: input.Trigger, Guidance: input.Guidance, Scope: input.Scope,
		Labels: input.Labels, Fingerprints: input.Fingerprints,
		Producer: input.Producer, Status: model.LearningNotePending,
	}
	if err := s.st.CreateLearningNote(ctx, note); err != nil {
		return nil, err
	}
	if err := s.recordTaskEvent(
		ctx,
		task.ID,
		"learning_note_captured",
		"Captured learning candidate",
		map[string]any{
			"learning_note_id": note.ID,
			"kind":             note.Kind,
			"trigger":          note.Trigger,
			"guidance":         note.Guidance,
		},
		note.CreatedAt,
	); err != nil {
		return nil, err
	}
	return s.st.GetLearningNote(ctx, note.ID)
}

func (s *Service) ListLearningNotes(
	ctx context.Context,
	input LearningNoteListInput,
) ([]model.LearningNote, error) {
	if input.Status != "" && !input.Status.Valid() {
		return nil, fmt.Errorf("invalid learning note status %q", input.Status)
	}
	if input.Limit < 0 || input.Limit > 200 {
		return nil, errors.New("limit must be between 0 and 200")
	}
	projectID := strings.TrimSpace(input.ProjectID)
	taskID := strings.TrimSpace(input.TaskID)
	if projectID == "" && taskID == "" {
		return nil, errors.New("project_id or task_id is required")
	}
	if projectID != "" {
		project, err := s.ResolveProject(ctx, projectID)
		if err != nil {
			return nil, err
		}
		projectID = project.ID
	}
	if taskID != "" {
		task, err := s.st.GetTask(ctx, taskID)
		if err != nil {
			return nil, err
		}
		if projectID != "" && projectID != task.ProjectID {
			return nil, fmt.Errorf("%w: task belongs to another project", store.ErrConflict)
		}
		projectID = task.ProjectID
	}
	return s.st.ListLearningNotes(ctx, store.LearningNoteFilter{
		ProjectID: projectID,
		TaskID:    taskID,
		Status:    input.Status,
		Limit:     input.Limit,
	})
}

func (s *Service) PromoteLearningNote(
	ctx context.Context,
	id string,
	agentName string,
	evidence string,
	memoryClasses ...model.MemoryClass,
) (*model.LearningNote, error) {
	evidence = strings.TrimSpace(evidence)
	if err := validateLearningText("evidence", evidence, true, maxLearningNoteEvidenceRunes); err != nil {
		return nil, err
	}
	memoryClass := model.MemoryClassExperience
	if len(memoryClasses) > 0 && memoryClasses[0] != "" {
		memoryClass = memoryClasses[0]
	}
	if !memoryClass.Valid() {
		return nil, fmt.Errorf("invalid memory class %q", memoryClass)
	}
	before, task, err := s.learningNoteAndTask(ctx, id, agentName)
	if err != nil {
		return nil, err
	}
	note, capsule, err := s.st.PromoteLearningNote(ctx, before.ID, evidence, memoryClass)
	if err != nil {
		return nil, err
	}
	if before.Status == model.LearningNotePending {
		if err := s.recordTaskEvent(
			ctx,
			task.ID,
			"learning_note_promoted",
			"Promoted verified learning candidate",
			map[string]any{
				"learning_note_id": note.ID,
				"capsule_id":       capsule.ID,
				"evidence":         note.Evidence,
				"memory_class":     capsule.MemoryClass,
			},
			note.ResolvedAt,
		); err != nil {
			return nil, err
		}
	}
	return note, nil
}

func (s *Service) RejectLearningNote(
	ctx context.Context,
	id string,
	agentName string,
	reason string,
) (*model.LearningNote, error) {
	reason = strings.TrimSpace(reason)
	if err := validateLearningText("reason", reason, true, maxLearningNoteRejectionRunes); err != nil {
		return nil, err
	}
	before, task, err := s.learningNoteAndTask(ctx, id, agentName)
	if err != nil {
		return nil, err
	}
	note, err := s.st.RejectLearningNote(ctx, before.ID, reason)
	if err != nil {
		return nil, err
	}
	if before.Status == model.LearningNotePending {
		if err := s.recordTaskEvent(
			ctx,
			task.ID,
			"learning_note_rejected",
			"Rejected learning candidate",
			map[string]any{
				"learning_note_id": note.ID,
				"reason":           note.RejectionReason,
			},
			note.ResolvedAt,
		); err != nil {
			return nil, err
		}
	}
	return note, nil
}

func (s *Service) UpdateLearningNote(
	ctx context.Context,
	id string,
	agentName string,
	input UpdateLearningNoteInput,
) (*model.LearningNote, error) {
	input.Trigger = strings.TrimSpace(input.Trigger)
	input.Guidance = strings.TrimSpace(input.Guidance)
	input.Scope = strings.TrimSpace(input.Scope)
	if err := validateLearningText("trigger", input.Trigger, true, maxLearningNoteTriggerRunes); err != nil {
		return nil, err
	}
	if err := validateLearningText("guidance", input.Guidance, true, maxLearningNoteGuidanceRunes); err != nil {
		return nil, err
	}
	if err := validateLearningText("scope", input.Scope, false, maxLearningNoteScopeRunes); err != nil {
		return nil, err
	}
	before, task, err := s.learningNoteAndTask(ctx, id, agentName)
	if err != nil {
		return nil, err
	}
	if before.Status != model.LearningNotePending {
		return nil, fmt.Errorf("%w: only pending learning notes can be edited", store.ErrConflict)
	}
	note, err := s.st.UpdateLearningNote(ctx, before.ID, input.Trigger, input.Guidance, input.Scope)
	if err != nil {
		return nil, err
	}
	if err := s.recordTaskEvent(
		ctx,
		task.ID,
		"learning_note_updated",
		"Updated learning candidate",
		map[string]any{
			"learning_note_id": note.ID,
			"before": map[string]any{
				"trigger": before.Trigger, "guidance": before.Guidance, "scope": before.Scope,
			},
			"after": map[string]any{
				"trigger": note.Trigger, "guidance": note.Guidance, "scope": note.Scope,
			},
		},
		note.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return note, nil
}

func (s *Service) learningNoteAndTask(
	ctx context.Context,
	id string,
	agentName string,
) (*model.LearningNote, *model.Task, error) {
	if strings.TrimSpace(agentName) == "" {
		return nil, nil, errors.New("agent identity required")
	}
	note, err := s.st.GetLearningNote(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, nil, err
	}
	task, err := s.st.GetTask(ctx, note.SourceTaskID)
	if err != nil {
		return nil, nil, err
	}
	return note, task, nil
}

func requireLiveOwner(task *model.Task, agentName string) error {
	agentName = strings.TrimSpace(agentName)
	if agentName == "" {
		return errors.New("agent identity required")
	}
	if task.Owner != agentName || task.LeaseExpiresAt <= time.Now().UnixMilli() {
		return fmt.Errorf(
			"%w: live source-task claim owned by %s required",
			store.ErrConflict,
			agentName,
		)
	}
	return nil
}

func validateLearningText(name, value string, required bool, maxRunes int) error {
	if required && value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return fmt.Errorf("%s exceeds %d characters", name, maxRunes)
	}
	return nil
}

func (s *Service) CreateCapsule(ctx context.Context, input CreateCapsuleInput) (*model.ExplorationCapsule, error) {
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.SourceTaskID = strings.TrimSpace(input.SourceTaskID)
	input.Trigger = strings.TrimSpace(input.Trigger)
	input.Title = strings.TrimSpace(input.Title)
	input.Summary = strings.TrimSpace(input.Summary)
	input.Evidence = strings.TrimSpace(input.Evidence)
	input.Producer = strings.TrimSpace(input.Producer)
	if input.Producer == "" {
		input.Producer = "codex"
	}
	if input.MemoryClass == "" {
		input.MemoryClass = model.MemoryClassExperience
	}
	if !input.MemoryClass.Valid() {
		return nil, fmt.Errorf("invalid memory class %q", input.MemoryClass)
	}
	if input.ProjectID == "" || input.Title == "" || input.Summary == "" || input.Evidence == "" {
		return nil, errors.New("project_id, title, summary, and evidence are required")
	}
	project, err := s.ResolveProject(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}
	capsule := &model.ExplorationCapsule{
		ProjectID: project.ID, SourceTaskID: input.SourceTaskID,
		MemoryClass: input.MemoryClass, Trigger: input.Trigger,
		Title: input.Title, Summary: input.Summary, Scope: strings.TrimSpace(input.Scope),
		Evidence: input.Evidence, Labels: input.Labels, Fingerprints: input.Fingerprints,
		Producer: input.Producer,
		Status:   model.CapsuleStatusActive,
	}
	if err := s.st.CreateCapsule(ctx, capsule); err != nil {
		return nil, err
	}
	return s.st.GetCapsule(ctx, capsule.ID)
}

func (s *Service) ListCapsules(ctx context.Context, input CapsuleListInput) ([]model.ExplorationCapsule, error) {
	if input.Status != "" && !input.Status.Valid() {
		return nil, fmt.Errorf("invalid capsule status %q", input.Status)
	}
	if input.MemoryClass != "" && !input.MemoryClass.Valid() {
		return nil, fmt.Errorf("invalid memory class %q", input.MemoryClass)
	}
	if input.Limit < 0 || input.Limit > 200 {
		return nil, errors.New("limit must be between 0 and 200")
	}
	project, err := s.ResolveProject(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}
	return s.st.ListCapsules(ctx, store.CapsuleFilter{
		ProjectID: project.ID, Query: input.Query, Status: input.Status,
		MemoryClass: input.MemoryClass, Limit: input.Limit,
	})
}

func (s *Service) GetCapsule(ctx context.Context, id string) (*model.ExplorationCapsule, error) {
	return s.st.GetCapsule(ctx, id)
}

func (s *Service) CreateMemoryRelation(
	ctx context.Context,
	sourceCapsuleID string,
	input CreateMemoryRelationInput,
) (*model.MemoryRelation, error) {
	if !input.Type.Valid() {
		return nil, fmt.Errorf("invalid memory relation type %q", input.Type)
	}
	if !input.TargetKind.Valid() {
		return nil, fmt.Errorf("invalid memory relation target kind %q", input.TargetKind)
	}
	input.TargetRef = strings.TrimSpace(input.TargetRef)
	input.Note = strings.TrimSpace(input.Note)
	if input.TargetRef == "" {
		return nil, errors.New("memory relation target cannot be blank")
	}
	if input.Type == model.MemoryRelationAppliesTo && input.TargetKind != model.MemoryRelationTargetScope {
		return nil, errors.New("applies-to relation requires scope target")
	}
	if (input.Type == model.MemoryRelationSupersedes || input.Type == model.MemoryRelationConflictsWith) &&
		input.TargetKind != model.MemoryRelationTargetCapsule {
		return nil, fmt.Errorf("%s relation requires capsule target", input.Type)
	}

	source, err := s.st.GetCapsule(ctx, sourceCapsuleID)
	if err != nil {
		return nil, err
	}
	relations, err := s.st.ListMemoryRelations(ctx, source.ProjectID)
	if err != nil {
		return nil, err
	}
	if input.TargetKind == model.MemoryRelationTargetCapsule {
		if input.TargetRef == sourceCapsuleID {
			return nil, errors.New("memory relation cannot target itself")
		}
		target, err := s.st.GetCapsule(ctx, input.TargetRef)
		if err != nil {
			return nil, err
		}
		if target.ProjectID != source.ProjectID {
			return nil, errors.New("memory relation cannot cross projects")
		}
		if input.Type == model.MemoryRelationSupersedes &&
			hasMemoryRelationPath(relations, input.TargetRef, sourceCapsuleID, model.MemoryRelationSupersedes) {
			return nil, errors.New("memory relation would create a supersedes cycle")
		}
		if input.Type == model.MemoryRelationConflictsWith &&
			hasDirectMemoryRelation(relations, input.TargetRef, sourceCapsuleID, model.MemoryRelationConflictsWith) {
			return nil, store.ErrConflict
		}
	}

	relation, err := s.st.CreateMemoryRelation(ctx, model.MemoryRelation{
		ProjectID:       source.ProjectID,
		SourceCapsuleID: sourceCapsuleID,
		Type:            input.Type,
		TargetKind:      input.TargetKind,
		TargetRef:       input.TargetRef,
		Note:            input.Note,
	})
	if err != nil {
		return nil, err
	}
	if input.Type == model.MemoryRelationSupersedes {
		stale := model.CapsuleStatusStale
		if _, err := s.st.UpdateCapsule(ctx, input.TargetRef, store.CapsuleUpdate{Status: &stale}); err != nil {
			_ = s.st.DeleteMemoryRelation(ctx, relation.ID)
			return nil, err
		}
	}
	return relation, nil
}

func (s *Service) DeleteMemoryRelation(ctx context.Context, id string) error {
	return s.st.DeleteMemoryRelation(ctx, id)
}

func hasDirectMemoryRelation(
	relations []model.MemoryRelation,
	sourceID, targetID string,
	relationType model.MemoryRelationType,
) bool {
	for _, relation := range relations {
		if relation.SourceCapsuleID == sourceID && relation.TargetKind == model.MemoryRelationTargetCapsule &&
			relation.TargetRef == targetID && relation.Type == relationType {
			return true
		}
	}
	return false
}

func hasMemoryRelationPath(
	relations []model.MemoryRelation,
	startID, targetID string,
	relationType model.MemoryRelationType,
) bool {
	visited := map[string]bool{}
	stack := []string{startID}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == targetID {
			return true
		}
		if visited[current] {
			continue
		}
		visited[current] = true
		for _, relation := range relations {
			if relation.SourceCapsuleID == current && relation.TargetKind == model.MemoryRelationTargetCapsule &&
				relation.Type == relationType {
				stack = append(stack, relation.TargetRef)
			}
		}
	}
	return false
}

func (s *Service) UpdateCapsule(ctx context.Context, id string, input UpdateCapsuleInput) (*model.ExplorationCapsule, error) {
	if input.Status != nil && !input.Status.Valid() {
		return nil, fmt.Errorf("invalid capsule status %q", *input.Status)
	}
	if input.MemoryClass != nil && !input.MemoryClass.Valid() {
		return nil, fmt.Errorf("invalid memory class %q", *input.MemoryClass)
	}
	for name, value := range map[string]*string{
		"title": input.Title, "summary": input.Summary, "evidence": input.Evidence,
	} {
		if value != nil {
			trimmed := strings.TrimSpace(*value)
			if trimmed == "" {
				return nil, fmt.Errorf("%s cannot be blank", name)
			}
			*value = trimmed
		}
	}
	return s.st.UpdateCapsule(ctx, id, store.CapsuleUpdate{
		MemoryClass: input.MemoryClass, Trigger: input.Trigger,
		Title: input.Title, Summary: input.Summary, Scope: input.Scope, Evidence: input.Evidence,
		Labels: input.Labels, Fingerprints: input.Fingerprints, Status: input.Status,
		ExpectedUpdatedAt: input.ExpectedUpdatedAt,
	})
}

func (s *Service) GetOrCreateTaskContext(ctx context.Context, taskID string) (*model.ContextSnapshot, error) {
	existing, err := s.st.GetContextSnapshot(ctx, taskID)
	if err == nil {
		recall := model.MemoryRecall{
			TaskID: existing.TaskID, ProjectID: existing.ProjectID,
			ProjectRules: existing.ProjectRules, SuggestedCapsules: existing.SuggestedCapsules,
			ContextRevision: existing.ContextRevision, Explanations: existing.Explanations,
		}
		if err := s.recordRecallReceipts(ctx, existing.Task, recall, "task-context"); err != nil {
			return nil, err
		}
		return existing, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	task, err := s.st.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(task.Owner) == "" || task.LeaseExpiresAt <= time.Now().UnixMilli() {
		return nil, fmt.Errorf("%w: live task claim required before creating context snapshot", store.ErrConflict)
	}
	capsules, err := s.st.ListCapsules(ctx, store.CapsuleFilter{
		ProjectID: task.ProjectID, Status: model.CapsuleStatusActive,
	})
	if err != nil {
		return nil, err
	}
	recall := buildMemoryRecall(*task, capsules, "")
	snapshot := &model.ContextSnapshot{
		TaskID: task.ID, ProjectID: task.ProjectID, Task: *task,
		ProjectRules: recall.ProjectRules, SuggestedCapsules: recall.SuggestedCapsules,
		ContextRevision: recall.ContextRevision, Explanations: recall.Explanations,
	}
	created, err := s.st.CreateContextSnapshot(ctx, snapshot)
	if err != nil {
		return nil, err
	}
	if err := s.recordRecallReceipts(ctx, *task, recall, "task-context"); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) RecallTaskMemory(
	ctx context.Context,
	taskID, agentName, query string,
) (*model.MemoryRecall, error) {
	task, err := s.st.GetTask(ctx, strings.TrimSpace(taskID))
	if err != nil {
		return nil, err
	}
	if err := requireLiveOwner(task, agentName); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("recall query required")
	}
	capsules, err := s.st.ListCapsules(ctx, store.CapsuleFilter{
		ProjectID: task.ProjectID, Status: model.CapsuleStatusActive,
	})
	if err != nil {
		return nil, err
	}
	recall := buildMemoryRecall(*task, capsules, query)
	if err := s.recordRecallReceipts(ctx, *task, recall, "dynamic-recall"); err != nil {
		return nil, err
	}
	return &recall, nil
}

func (s *Service) RecordCapsuleUsage(ctx context.Context, input RecordUsageInput) (*model.CapsuleUsage, error) {
	if !input.Outcome.Valid() {
		return nil, fmt.Errorf("invalid capsule outcome %q", input.Outcome)
	}
	state := model.MemoryImpactApplied
	switch input.Outcome {
	case model.CapsuleOutcomeHelpful:
		state = model.MemoryImpactHelpful
	case model.CapsuleOutcomeRejected:
		state = model.MemoryImpactRejected
	case model.CapsuleOutcomeStale:
		state = model.MemoryImpactStale
	}
	evidence := input.Evidence
	if state.Terminal() && len(evidence) == 0 {
		evidence = []model.MemoryImpactEvidence{{
			Kind: "observation", Ref: "legacy-capsule-usage",
			Summary: strings.TrimSpace(input.Notes),
		}}
	}
	input.Evidence = evidence
	if _, err := s.RecordCapsuleImpact(ctx, input, state, 0); err != nil {
		return nil, err
	}
	return s.st.UpsertCapsuleUsage(ctx, &model.CapsuleUsage{
		CapsuleID: input.CapsuleID, TaskID: input.TaskID,
		Outcome: input.Outcome, Notes: strings.TrimSpace(input.Notes),
	})
}

func (s *Service) GetLearningMetrics(ctx context.Context, projectID string) (*model.LearningMetrics, error) {
	project, err := s.ResolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	metrics, err := s.st.GetLearningMetrics(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	runMetrics, err := s.st.GetRunMetrics(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	metrics.RunCount = runMetrics.RunCount
	metrics.CompletedRunCount = runMetrics.CompletedRunCount
	metrics.ActiveRunCount = runMetrics.ActiveRunCount
	metrics.BlockedRunCount = runMetrics.BlockedRunCount
	metrics.ResumedRunCount = runMetrics.ResumedRunCount
	if metrics.RunCount > 0 {
		metrics.RunCompletionRate = float64(metrics.CompletedRunCount) / float64(metrics.RunCount)
	}
	if metrics.BlockedRunCount > 0 {
		metrics.RecoveryRate = float64(runMetrics.RecoveredRunCount) / float64(metrics.BlockedRunCount)
	}
	return metrics, nil
}

type rankedCapsule struct {
	capsule model.ExplorationCapsule
	score   float64
}

func buildMemoryRecall(task model.Task, capsules []model.ExplorationCapsule, query string) model.MemoryRecall {
	recallTask := task
	if query != "" {
		recallTask.Description = strings.TrimSpace(task.Description + "\n" + query)
	}
	projectRules := selectProjectRules(capsules, projectRuleBudgetRunes, maxProjectRules)
	suggestedCapsules := matchCapsules(
		recallTask,
		capsules,
		experienceRecallBudgetRunes,
		maxRecalledExperiences,
	)
	return model.MemoryRecall{
		TaskID: task.ID, ProjectID: task.ProjectID, Query: query,
		ProjectRules: projectRules, SuggestedCapsules: suggestedCapsules,
		ContextRevision: memoryContextRevision(projectRules, suggestedCapsules),
		Explanations:    explainMemoryRecall(recallTask, projectRules, suggestedCapsules),
		RecalledAt:      time.Now().UnixMilli(),
	}
}

func selectProjectRules(capsules []model.ExplorationCapsule, budgetRunes, maxCount int) []model.ExplorationCapsule {
	rules := make([]model.ExplorationCapsule, 0)
	for _, capsule := range capsules {
		if capsule.Status == model.CapsuleStatusActive &&
			capsule.MemoryClass == model.MemoryClassProjectRule &&
			capsule.Validation != model.MemoryValidationDisputed {
			rules = append(rules, capsule)
		}
	}
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Confidence != rules[j].Confidence {
			return rules[i].Confidence > rules[j].Confidence
		}
		if rules[i].UpdatedAt != rules[j].UpdatedAt {
			return rules[i].UpdatedAt > rules[j].UpdatedAt
		}
		return rules[i].ID < rules[j].ID
	})
	return fitMemoryBudget(rules, budgetRunes, maxCount)
}

func matchCapsules(
	task model.Task,
	capsules []model.ExplorationCapsule,
	budgetRunes, maxCount int,
) []model.ExplorationCapsule {
	taskTitle := tokenSet(task.Title)
	taskBody := tokenSet(task.Description)
	taskLabels := stringSet(task.Labels)
	taskAll := mergeSets(taskTitle, taskBody, taskLabels)
	ranked := make([]rankedCapsule, 0, len(capsules))
	for _, capsule := range capsules {
		if capsule.Status != model.CapsuleStatusActive ||
			capsule.MemoryClass == model.MemoryClassProjectRule ||
			capsule.Validation == model.MemoryValidationDisputed {
			continue
		}
		relevance := 5*overlap(taskLabels, stringSet(capsule.Labels)) +
			4*overlap(taskAll, stringSet(capsule.Fingerprints)) +
			3*overlap(taskTitle, tokenSet(capsule.Title)) +
			2*overlap(taskAll, tokenSet(capsule.Summary)) +
			2*overlap(taskAll, tokenSet(capsule.Trigger)) +
			overlap(taskAll, tokenSet(capsule.Scope))
		for _, relation := range capsule.Relations {
			if relation.Direction == model.MemoryRelationOutgoing &&
				relation.Type == model.MemoryRelationAppliesTo &&
				overlap(taskAll, tokenSet(relation.TargetRef)) > 0 {
				relevance += 6
			}
		}
		if relevance > 0 {
			score := float64(relevance) + 2*capsule.Confidence +
				0.25*float64(capsule.HelpfulCount) - 0.25*float64(capsule.RejectedCount)
			ranked = append(ranked, rankedCapsule{capsule: capsule, score: score})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		if ranked[i].capsule.UpdatedAt != ranked[j].capsule.UpdatedAt {
			return ranked[i].capsule.UpdatedAt > ranked[j].capsule.UpdatedAt
		}
		return ranked[i].capsule.ID < ranked[j].capsule.ID
	})
	out := make([]model.ExplorationCapsule, len(ranked))
	for i := range ranked {
		out[i] = ranked[i].capsule
	}
	return fitMemoryBudget(out, budgetRunes, maxCount)
}

func explainMemoryRecall(
	task model.Task,
	projectRules, suggested []model.ExplorationCapsule,
) []model.MemoryRecallExplanation {
	out := make([]model.MemoryRecallExplanation, 0, len(projectRules)+len(suggested))
	for _, capsule := range projectRules {
		explanation := explainCapsuleMatch(task, capsule)
		explanation.Reasons = append(
			[]model.MemoryRecallReason{{Code: "project-rule"}},
			explanation.Reasons...,
		)
		out = append(out, explanation)
	}
	for _, capsule := range suggested {
		out = append(out, explainCapsuleMatch(task, capsule))
	}
	return out
}

func explainCapsuleMatch(task model.Task, capsule model.ExplorationCapsule) model.MemoryRecallExplanation {
	taskTitle := tokenSet(task.Title)
	taskBody := tokenSet(task.Description)
	taskLabels := stringSet(task.Labels)
	taskAll := mergeSets(taskTitle, taskBody, taskLabels)
	explanation := model.MemoryRecallExplanation{
		CapsuleID: capsule.ID,
		Reasons:   []model.MemoryRecallReason{},
		Warnings:  []string{},
	}
	addReason := func(code, value string, weight int) {
		explanation.Reasons = append(explanation.Reasons, model.MemoryRecallReason{Code: code, Value: value})
		explanation.Score += float64(weight)
	}
	if value := joinedOverlap(taskLabels, stringSet(capsule.Labels)); value != "" {
		addReason("label-match", value, 5)
	}
	if value := joinedOverlap(taskAll, stringSet(capsule.Fingerprints)); value != "" {
		addReason("fingerprint-match", value, 4)
	}
	if value := joinedOverlap(taskTitle, tokenSet(capsule.Title)); value != "" {
		addReason("title-match", value, 3)
	}
	if value := joinedOverlap(taskAll, tokenSet(capsule.Summary)); value != "" {
		addReason("summary-match", value, 2)
	}
	if value := joinedOverlap(taskAll, tokenSet(capsule.Trigger)); value != "" {
		addReason("trigger-match", value, 2)
	}
	if value := joinedOverlap(taskAll, tokenSet(capsule.Scope)); value != "" {
		addReason("scope-match", value, 1)
	}
	for _, relation := range capsule.Relations {
		switch relation.Type {
		case model.MemoryRelationAppliesTo:
			if relation.Direction == model.MemoryRelationOutgoing &&
				overlap(taskAll, tokenSet(relation.TargetRef)) > 0 {
				addReason("applies-to", relation.TargetRef, 6)
			}
		case model.MemoryRelationValidatedBy:
			if relation.Direction == model.MemoryRelationOutgoing {
				explanation.Reasons = append(explanation.Reasons, model.MemoryRecallReason{
					Code: "validated-by", Value: relation.TargetRef,
				})
			}
		case model.MemoryRelationConflictsWith:
			conflictID := relation.TargetRef
			if relation.Direction == model.MemoryRelationIncoming {
				conflictID = relation.SourceCapsuleID
			}
			explanation.Warnings = append(explanation.Warnings, "conflicts-with:"+conflictID)
		}
	}
	explanation.Score += 2*capsule.Confidence +
		0.25*float64(capsule.HelpfulCount) - 0.25*float64(capsule.RejectedCount)
	return explanation
}

func joinedOverlap(left, right map[string]struct{}) string {
	values := make([]string, 0)
	for value := range left {
		if _, ok := right[value]; ok {
			values = append(values, value)
		}
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

func memoryContextRevision(groups ...[]model.ExplorationCapsule) string {
	parts := make([]string, 0)
	for _, capsules := range groups {
		for _, capsule := range capsules {
			parts = append(parts, fmt.Sprintf("%s:%d", capsule.ID, capsule.UpdatedAt))
			for _, relation := range capsule.Relations {
				parts = append(parts, relation.ID)
			}
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return fmt.Sprintf("%x", sum[:8])
}

func fitMemoryBudget(capsules []model.ExplorationCapsule, budgetRunes, maxCount int) []model.ExplorationCapsule {
	out := make([]model.ExplorationCapsule, 0, len(capsules))
	used := 0
	for _, capsule := range capsules {
		if maxCount > 0 && len(out) >= maxCount {
			break
		}
		cost := utf8.RuneCountInString(
			capsule.Trigger + capsule.Title + capsule.Summary + capsule.Scope + capsule.Evidence,
		)
		if len(out) > 0 && budgetRunes > 0 && used+cost > budgetRunes {
			continue
		}
		out = append(out, capsule)
		used += cost
	}
	return out
}

func tokenSet(value string) map[string]struct{} {
	return stringSet(strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-'
	}))
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.ToLower(strings.TrimSpace(value)); value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func mergeSets(sets ...map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{})
	for _, set := range sets {
		for value := range set {
			out[value] = struct{}{}
		}
	}
	return out
}

func overlap(left, right map[string]struct{}) int {
	count := 0
	for value := range left {
		if _, ok := right[value]; ok {
			count++
		}
	}
	return count
}
