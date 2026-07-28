package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

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
