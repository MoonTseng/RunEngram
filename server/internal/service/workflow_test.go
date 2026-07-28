package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"taskline_server/api/model"
	"taskline_server/internal/service"
	"taskline_server/internal/store"
)

type fakePullRequestVerifier struct {
	calls int
}

func (f *fakePullRequestVerifier) VerifyPullRequest(_ context.Context, _ service.PullRequestRef) (service.PullRequestStatus, error) {
	f.calls++
	return service.PullRequestStatus{}, nil
}

func newWorkflowSvc(t *testing.T, verifier service.PullRequestVerifier) *service.Service {
	t.Helper()
	st, err := store.New(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = st.Close() })
	return service.New(st, service.WithPullRequestVerifier(verifier))
}

func newWorkflowTask(t *testing.T, s *service.Service) *model.Task {
	t.Helper()
	ctx := context.Background()
	p, err := s.CreateProject(ctx, "workflow", "")
	require.NoError(t, err)
	task, err := s.CreateTask(ctx, p.ID, "guard transitions", "", model.TaskTypeFeature, 0, true)
	require.NoError(t, err)
	return task
}

func TestReviewAndDoneAllowManualTransitionWithoutPullRequestEvidence(t *testing.T) {
	ctx := context.Background()
	verifier := &fakePullRequestVerifier{}
	s := newWorkflowSvc(t, verifier)
	task := newWorkflowTask(t, s)

	review := model.StateReview
	updated, err := s.UpdateTask(ctx, task.ID, store.TaskUpdate{State: &review})
	require.NoError(t, err)
	require.Equal(t, model.StateReview, updated.State)

	done := model.StateDone
	updated, err = s.UpdateTask(ctx, task.ID, store.TaskUpdate{State: &done})
	require.NoError(t, err)
	require.Equal(t, model.StateDone, updated.State)
	require.Zero(t, verifier.calls)
}

func TestSameStateUpdateDoesNotReverifyPullRequest(t *testing.T) {
	ctx := context.Background()
	verifier := &fakePullRequestVerifier{}
	s := newWorkflowSvc(t, verifier)
	task := newWorkflowTask(t, s)

	start := model.StateStart
	title := "renamed"
	updated, err := s.UpdateTask(ctx, task.ID, store.TaskUpdate{State: &start, Title: &title})
	require.NoError(t, err)
	require.Equal(t, title, updated.Title)
	require.Zero(t, verifier.calls)
}

func TestParsePullRequestURL(t *testing.T) {
	ref, ok := service.ParsePullRequestURL("https://github.com/octo-org/example-repo/pull/42/files?diff=split")
	require.True(t, ok)
	require.Equal(t, "octo-org", ref.Owner)
	require.Equal(t, "example-repo", ref.Repository)
	require.Equal(t, 42, ref.Number)
	require.Equal(t, "https://github.com/octo-org/example-repo/pull/42", ref.URL)

	for _, raw := range []string{
		"http://github.com/octo-org/example-repo/pull/42",
		"https://example.com/octo-org/example-repo/pull/42",
		"https://github.com/octo-org/example-repo/issues/42",
		"https://github.com/octo-org/example-repo/pull/not-a-number",
	} {
		_, ok := service.ParsePullRequestURL(raw)
		require.False(t, ok, raw)
	}
}
