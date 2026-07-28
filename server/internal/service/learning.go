package service

import (
	"context"
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
	Title        string
	Summary      string
	Scope        string
	Evidence     string
	Labels       []string
	Fingerprints []string
	Producer     string
}

type UpdateCapsuleInput struct {
	Title        *string
	Summary      *string
	Scope        *string
	Evidence     *string
	Labels       *[]string
	Fingerprints *[]string
	Status       *model.CapsuleStatus
}

type CapsuleListInput struct {
	ProjectID string
	Query     string
	Status    model.CapsuleStatus
	Limit     int
}

type RecordUsageInput struct {
	CapsuleID string
	TaskID    string
	Outcome   model.CapsuleOutcome
	Notes     string
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

const (
	maxLearningNoteTriggerRunes   = 2_000
	maxLearningNoteGuidanceRunes  = 8_000
	maxLearningNoteScopeRunes     = 2_000
	maxLearningNoteEvidenceRunes  = 32_000
	maxLearningNoteRejectionRunes = 2_000
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
) (*model.LearningNote, error) {
	evidence = strings.TrimSpace(evidence)
	if err := validateLearningText("evidence", evidence, true, maxLearningNoteEvidenceRunes); err != nil {
		return nil, err
	}
	before, task, err := s.learningNoteAndOwnedTask(ctx, id, agentName)
	if err != nil {
		return nil, err
	}
	note, capsule, err := s.st.PromoteLearningNote(ctx, before.ID, evidence)
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
	before, task, err := s.learningNoteAndOwnedTask(ctx, id, agentName)
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

func (s *Service) learningNoteAndOwnedTask(
	ctx context.Context,
	id string,
	agentName string,
) (*model.LearningNote, *model.Task, error) {
	note, err := s.st.GetLearningNote(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, nil, err
	}
	task, err := s.st.GetTask(ctx, note.SourceTaskID)
	if err != nil {
		return nil, nil, err
	}
	if err := requireLiveOwner(task, agentName); err != nil {
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
	input.Title = strings.TrimSpace(input.Title)
	input.Summary = strings.TrimSpace(input.Summary)
	input.Evidence = strings.TrimSpace(input.Evidence)
	input.Producer = strings.TrimSpace(input.Producer)
	if input.Producer == "" {
		input.Producer = "codex"
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
	if input.Limit < 0 || input.Limit > 200 {
		return nil, errors.New("limit must be between 0 and 200")
	}
	project, err := s.ResolveProject(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}
	return s.st.ListCapsules(ctx, store.CapsuleFilter{
		ProjectID: project.ID, Query: input.Query, Status: input.Status, Limit: input.Limit,
	})
}

func (s *Service) GetCapsule(ctx context.Context, id string) (*model.ExplorationCapsule, error) {
	return s.st.GetCapsule(ctx, id)
}

func (s *Service) UpdateCapsule(ctx context.Context, id string, input UpdateCapsuleInput) (*model.ExplorationCapsule, error) {
	if input.Status != nil && !input.Status.Valid() {
		return nil, fmt.Errorf("invalid capsule status %q", *input.Status)
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
		Title: input.Title, Summary: input.Summary, Scope: input.Scope, Evidence: input.Evidence,
		Labels: input.Labels, Fingerprints: input.Fingerprints, Status: input.Status,
	})
}

func (s *Service) GetOrCreateTaskContext(ctx context.Context, taskID string) (*model.ContextSnapshot, error) {
	existing, err := s.st.GetContextSnapshot(ctx, taskID)
	if err == nil {
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
	snapshot := &model.ContextSnapshot{
		TaskID: task.ID, ProjectID: task.ProjectID, Task: *task,
		SuggestedCapsules: matchCapsules(*task, capsules, 5),
	}
	return s.st.CreateContextSnapshot(ctx, snapshot)
}

func (s *Service) RecordCapsuleUsage(ctx context.Context, input RecordUsageInput) (*model.CapsuleUsage, error) {
	if !input.Outcome.Valid() {
		return nil, fmt.Errorf("invalid capsule outcome %q", input.Outcome)
	}
	capsule, err := s.st.GetCapsule(ctx, input.CapsuleID)
	if err != nil {
		return nil, err
	}
	task, err := s.st.GetTask(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}
	if capsule.ProjectID != task.ProjectID {
		return nil, fmt.Errorf("%w: capsule and task belong to different projects", store.ErrConflict)
	}
	usage, err := s.st.UpsertCapsuleUsage(ctx, &model.CapsuleUsage{
		CapsuleID: capsule.ID, TaskID: task.ID, Outcome: input.Outcome, Notes: strings.TrimSpace(input.Notes),
	})
	if err != nil {
		return nil, err
	}
	if input.Outcome == model.CapsuleOutcomeStale {
		status := model.CapsuleStatusStale
		if _, err := s.st.UpdateCapsule(ctx, capsule.ID, store.CapsuleUpdate{Status: &status}); err != nil {
			return nil, err
		}
	}
	return usage, nil
}

func (s *Service) GetLearningMetrics(ctx context.Context, projectID string) (*model.LearningMetrics, error) {
	project, err := s.ResolveProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.st.GetLearningMetrics(ctx, project.ID)
}

type rankedCapsule struct {
	capsule model.ExplorationCapsule
	score   int
}

func matchCapsules(task model.Task, capsules []model.ExplorationCapsule, limit int) []model.ExplorationCapsule {
	taskTitle := tokenSet(task.Title)
	taskBody := tokenSet(task.Description)
	taskLabels := stringSet(task.Labels)
	taskAll := mergeSets(taskTitle, taskBody, taskLabels)
	ranked := make([]rankedCapsule, 0, len(capsules))
	for _, capsule := range capsules {
		if capsule.Status != model.CapsuleStatusActive {
			continue
		}
		score := 5*overlap(taskLabels, stringSet(capsule.Labels)) +
			4*overlap(taskAll, stringSet(capsule.Fingerprints)) +
			3*overlap(taskTitle, tokenSet(capsule.Title)) +
			2*overlap(taskAll, tokenSet(capsule.Summary)) +
			overlap(taskAll, tokenSet(capsule.Scope))
		if score > 0 {
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
	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]model.ExplorationCapsule, len(ranked))
	for i := range ranked {
		out[i] = ranked[i].capsule
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
