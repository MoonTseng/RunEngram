package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"taskline_server/api/model"
)

var (
	// ErrStateEntryBlocked means the task is missing required workflow evidence.
	ErrStateEntryBlocked = errors.New("state entry blocked")
	// ErrStateEntryVerificationUnavailable means external evidence could not be checked.
	ErrStateEntryVerificationUnavailable = errors.New("state entry verification unavailable")
	// ErrPullRequestNotFound distinguishes an invalid PR artifact from an integration outage.
	ErrPullRequestNotFound = errors.New("pull request not found")
)

const (
	PullRequestOpen   = "OPEN"
	PullRequestClosed = "CLOSED"
	PullRequestMerged = "MERGED"

	CheckRollupSuccess  = "SUCCESS"
	CheckRollupPending  = "PENDING"
	CheckRollupFailure  = "FAILURE"
	CheckRollupError    = "ERROR"
	CheckRollupExpected = "EXPECTED"
)

// PullRequestRef is the canonical identity parsed from a linked GitHub PR.
type PullRequestRef struct {
	URL        string
	Owner      string
	Repository string
	Number     int
}

// PullRequestStatus contains only the external facts needed by workflow rules.
type PullRequestStatus struct {
	State                   string
	Merged                  bool
	UnresolvedReviewThreads int
	CheckRollupState        string
}

// PullRequestVerifier keeps GitHub-specific API details outside the service.
type PullRequestVerifier interface {
	VerifyPullRequest(context.Context, PullRequestRef) (PullRequestStatus, error)
}

// StateEntryRule validates evidence before a task can enter a target state.
type StateEntryRule interface {
	ValidateStateEntry(context.Context, *model.Task) error
}

// StateEntryRuleFunc adapts a function into a StateEntryRule.
type StateEntryRuleFunc func(context.Context, *model.Task) error

func (f StateEntryRuleFunc) ValidateStateEntry(ctx context.Context, task *model.Task) error {
	return f(ctx, task)
}

type unavailablePullRequestVerifier struct{}

func (unavailablePullRequestVerifier) VerifyPullRequest(context.Context, PullRequestRef) (PullRequestStatus, error) {
	return PullRequestStatus{}, errors.New("GitHub verifier is not configured")
}

func defaultStateEntryRules(_ PullRequestVerifier) map[model.TaskState][]StateEntryRule {
	return make(map[model.TaskState][]StateEntryRule)
}

func (s *Service) validateStateEntry(ctx context.Context, task *model.Task, target model.TaskState) error {
	for _, rule := range s.stateEntryRules[target] {
		if err := rule.ValidateStateEntry(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

// ParsePullRequestURL accepts canonical HTTPS GitHub pull request URLs.
func ParsePullRequestURL(raw string) (PullRequestRef, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") {
		return PullRequestRef{}, false
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return PullRequestRef{}, false
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[1] == "" || parts[2] != "pull" {
		return PullRequestRef{}, false
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return PullRequestRef{}, false
	}
	return PullRequestRef{
		URL:        fmt.Sprintf("https://github.com/%s/%s/pull/%d", parts[0], parts[1], number),
		Owner:      parts[0],
		Repository: parts[1],
		Number:     number,
	}, true
}
