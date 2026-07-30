package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Client is a thin HTTP wrapper for runengram-server.
type Client struct {
	BaseURL string
	HTTP    *http.Client
	Token   string
}

// New constructs a Client targeting baseURL.
func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Agent mirrors the server-side agent identity shape.
type Agent struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type ActiveClaim struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	ClaimedAt      int64  `json:"claimed_at"`
	ClaimedForMS   int64  `json:"claimed_for_ms"`
	LeaseExpiresAt int64  `json:"lease_expires_at"`
}

type ServerStatus struct {
	OK          bool          `json:"ok"`
	ServerTime  int64         `json:"server_time"`
	Agent       *Agent        `json:"agent,omitempty"`
	ActiveTasks []ActiveClaim `json:"active_tasks"`
}

// Project mirrors the server-side project shape.
type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// Task mirrors the server-side task shape.
type Task struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"project_id"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Type           string         `json:"type"`
	State          string         `json:"state"`
	Priority       int            `json:"priority"`
	Labels         []string       `json:"labels"`
	Owner          string         `json:"owner"`
	ClaimedAt      int64          `json:"claimed_at"`
	LeaseExpiresAt int64          `json:"lease_expires_at"`
	CompletedAt    int64          `json:"completed_at"`
	DependsOn      []string       `json:"depends_on,omitempty"`
	Images         []Image        `json:"images,omitempty"`
	Docs           []Doc          `json:"docs,omitempty"`
	Links          []Link         `json:"links,omitempty"`
	LearningNotes  []LearningNote `json:"learning_notes,omitempty"`
	CreatedAt      int64          `json:"created_at"`
	UpdatedAt      int64          `json:"updated_at"`
}

type LearningNote struct {
	ID              string   `json:"id"`
	ProjectID       string   `json:"project_id"`
	SourceTaskID    string   `json:"source_task_id"`
	Kind            string   `json:"kind"`
	Trigger         string   `json:"trigger"`
	Guidance        string   `json:"guidance"`
	Scope           string   `json:"scope"`
	Labels          []string `json:"labels"`
	Fingerprints    []string `json:"fingerprints"`
	Producer        string   `json:"producer"`
	Status          string   `json:"status"`
	Evidence        string   `json:"evidence"`
	CapsuleID       string   `json:"capsule_id"`
	RejectionReason string   `json:"rejection_reason"`
	CreatedAt       int64    `json:"created_at"`
	UpdatedAt       int64    `json:"updated_at"`
	ResolvedAt      int64    `json:"resolved_at"`
}

type ExplorationCapsule struct {
	ID            string           `json:"id"`
	ProjectID     string           `json:"project_id"`
	SourceTaskID  string           `json:"source_task_id"`
	MemoryClass   string           `json:"memory_class"`
	Trigger       string           `json:"trigger"`
	Title         string           `json:"title"`
	Summary       string           `json:"summary"`
	Scope         string           `json:"scope"`
	Evidence      string           `json:"evidence"`
	Labels        []string         `json:"labels"`
	Fingerprints  []string         `json:"fingerprints"`
	Producer      string           `json:"producer"`
	Status        string           `json:"status"`
	Validation    string           `json:"validation"`
	Confidence    float64          `json:"confidence"`
	UseCount      int              `json:"use_count"`
	HelpfulCount  int              `json:"helpful_count"`
	RejectedCount int              `json:"rejected_count"`
	Relations     []MemoryRelation `json:"relations"`
	CreatedAt     int64            `json:"created_at"`
	UpdatedAt     int64            `json:"updated_at"`
}

type MemoryRelation struct {
	ID              string `json:"id"`
	ProjectID       string `json:"project_id"`
	SourceCapsuleID string `json:"source_capsule_id"`
	Type            string `json:"type"`
	TargetKind      string `json:"target_kind"`
	TargetRef       string `json:"target_ref"`
	Note            string `json:"note"`
	Direction       string `json:"direction"`
	CreatedAt       int64  `json:"created_at"`
}

type MemoryRecallReason struct {
	Code  string `json:"code"`
	Value string `json:"value"`
}

type MemoryRecallExplanation struct {
	CapsuleID string               `json:"capsule_id"`
	Score     float64              `json:"score"`
	Reasons   []MemoryRecallReason `json:"reasons"`
	Warnings  []string             `json:"warnings"`
}

type ContextSnapshot struct {
	ID                string                    `json:"id"`
	TaskID            string                    `json:"task_id"`
	ProjectID         string                    `json:"project_id"`
	Task              Task                      `json:"task"`
	ProjectRules      []ExplorationCapsule      `json:"project_rules"`
	SuggestedCapsules []ExplorationCapsule      `json:"suggested_capsules"`
	ContextRevision   string                    `json:"context_revision"`
	Explanations      []MemoryRecallExplanation `json:"explanations"`
	CreatedAt         int64                     `json:"created_at"`
}

type MemoryRecall struct {
	TaskID            string                    `json:"task_id"`
	ProjectID         string                    `json:"project_id"`
	Query             string                    `json:"query"`
	ProjectRules      []ExplorationCapsule      `json:"project_rules"`
	SuggestedCapsules []ExplorationCapsule      `json:"suggested_capsules"`
	ContextRevision   string                    `json:"context_revision"`
	Explanations      []MemoryRecallExplanation `json:"explanations"`
	RecalledAt        int64                     `json:"recalled_at"`
}

type CapsuleUsage struct {
	ID        string `json:"id"`
	CapsuleID string `json:"capsule_id"`
	TaskID    string `json:"task_id"`
	Outcome   string `json:"outcome"`
	Notes     string `json:"notes"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type MemoryImpactEvidence struct {
	Kind    string `json:"kind"`
	Ref     string `json:"ref"`
	Summary string `json:"summary"`
}

type MemoryImpact struct {
	ID              string                 `json:"id"`
	ProjectID       string                 `json:"project_id"`
	TaskID          string                 `json:"task_id"`
	CapsuleID       string                 `json:"capsule_id"`
	State           string                 `json:"state"`
	RecallSource    string                 `json:"recall_source"`
	ContextRevision string                 `json:"context_revision"`
	RecallScore     float64                `json:"recall_score"`
	RecallReasons   []string               `json:"recall_reasons"`
	Stage           string                 `json:"stage"`
	Notes           string                 `json:"notes"`
	Evidence        []MemoryImpactEvidence `json:"evidence"`
	Actor           string                 `json:"actor"`
	CreatedAt       int64                  `json:"created_at"`
	UpdatedAt       int64                  `json:"updated_at"`
	ResolvedAt      int64                  `json:"resolved_at"`
}

type RecordCapsuleUsageInput struct {
	TaskID            string                 `json:"task_id"`
	Outcome           string                 `json:"outcome"`
	Stage             string                 `json:"stage,omitempty"`
	Notes             string                 `json:"notes"`
	Evidence          []MemoryImpactEvidence `json:"evidence,omitempty"`
	ExpectedUpdatedAt int64                  `json:"expected_updated_at,omitempty"`
}

type LearningMetrics struct {
	CapsuleCount       int     `json:"capsule_count"`
	ActiveCapsuleCount int     `json:"active_capsule_count"`
	LearningNoteCount  int     `json:"learning_note_count"`
	PendingNoteCount   int     `json:"pending_note_count"`
	PromotedNoteCount  int     `json:"promoted_note_count"`
	RejectedNoteCount  int     `json:"rejected_note_count"`
	SnapshotTaskCount  int     `json:"snapshot_task_count"`
	ReusedTaskCount    int     `json:"reused_task_count"`
	HelpfulCount       int     `json:"helpful_count"`
	RejectedCount      int     `json:"rejected_count"`
	StaleCount         int     `json:"stale_count"`
	HelpfulRate        float64 `json:"helpful_rate"`
	PromotionRate      float64 `json:"promotion_rate"`
	RunCount           int     `json:"run_count"`
	CompletedRunCount  int     `json:"completed_run_count"`
	ActiveRunCount     int     `json:"active_run_count"`
	BlockedRunCount    int     `json:"blocked_run_count"`
	ResumedRunCount    int     `json:"resumed_run_count"`
	RunCompletionRate  float64 `json:"run_completion_rate"`
	RecoveryRate       float64 `json:"recovery_rate"`
}

type CaptureLearningNoteInput struct {
	SourceTaskID string   `json:"source_task_id"`
	Kind         string   `json:"kind"`
	Trigger      string   `json:"trigger"`
	Guidance     string   `json:"guidance"`
	Scope        string   `json:"scope,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	Fingerprints []string `json:"fingerprints,omitempty"`
	Producer     string   `json:"producer,omitempty"`
}

type UpdateLearningNoteInput struct {
	Trigger  string `json:"trigger"`
	Guidance string `json:"guidance"`
	Scope    string `json:"scope,omitempty"`
}

type CreateCapsuleInput struct {
	SourceTaskID string   `json:"source_task_id,omitempty"`
	MemoryClass  string   `json:"memory_class,omitempty"`
	Trigger      string   `json:"trigger,omitempty"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Scope        string   `json:"scope,omitempty"`
	Evidence     string   `json:"evidence"`
	Labels       []string `json:"labels,omitempty"`
	Fingerprints []string `json:"fingerprints,omitempty"`
	Producer     string   `json:"producer,omitempty"`
}

type UpdateCapsuleInput struct {
	Title             *string `json:"title,omitempty"`
	Summary           *string `json:"summary,omitempty"`
	Trigger           *string `json:"trigger,omitempty"`
	Scope             *string `json:"scope,omitempty"`
	Evidence          *string `json:"evidence,omitempty"`
	ExpectedUpdatedAt int64   `json:"expected_updated_at"`
}

type CreateMemoryRelationInput struct {
	Type       string `json:"type"`
	TargetKind string `json:"target_kind"`
	TargetRef  string `json:"target_ref"`
	Note       string `json:"note,omitempty"`
}

// TaskEvent is one append-only task operation record.
type TaskEvent struct {
	ID        string         `json:"id"`
	TaskID    string         `json:"task_id"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Summary   string         `json:"summary"`
	Details   map[string]any `json:"details"`
	CreatedAt int64          `json:"created_at"`
}

// Link is a URL attached to a task.
type Link struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	URL       string `json:"url"`
	Label     string `json:"label"`
	CreatedAt int64  `json:"created_at"`
}

// Image is an attachment record.
type Image struct {
	ID         string `json:"id"`
	TaskID     string `json:"task_id"`
	Filename   string `json:"filename"`
	MimeType   string `json:"mime_type"`
	SizeBytes  int64  `json:"size_bytes"`
	URL        string `json:"url,omitempty"`
	UploadedAt int64  `json:"uploaded_at"`
}

// Doc is a markdown document attached to a task.
type Doc struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	Title     string `json:"title"`
	URL       string `json:"url,omitempty"`
	Content   string `json:"content,omitempty"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// ─── Agent endpoints ────────────────────────────────────────────────────

type RegisterAgentInput struct {
	Name string `json:"name"`
}

type RegisterAgentOutput struct {
	Agent Agent  `json:"agent"`
	Token string `json:"token"`
}

func (c *Client) RegisterAgent(in RegisterAgentInput) (*RegisterAgentOutput, error) {
	var out RegisterAgentOutput
	if err := c.do("POST", "/api/v1/agents/register", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetStatus() (*ServerStatus, error) {
	var out ServerStatus
	if err := c.do("GET", "/api/v1/status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ─── Project endpoints ──────────────────────────────────────────────────

type CreateProjectInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (c *Client) CreateProject(in CreateProjectInput) (*Project, error) {
	var out Project
	if err := c.do("POST", "/api/v1/projects", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type listProjectsResp struct {
	Projects []Project `json:"projects"`
}

func (c *Client) ListProjects() ([]Project, error) {
	var out listProjectsResp
	if err := c.do("GET", "/api/v1/projects", nil, &out); err != nil {
		return nil, err
	}
	return out.Projects, nil
}

func (c *Client) DeleteProject(idOrName string) error {
	return c.do("DELETE", "/api/v1/projects/"+url.PathEscape(idOrName), nil, nil)
}

// ─── Task endpoints ─────────────────────────────────────────────────────

type CreateTaskInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Priority    int      `json:"priority"`
	Labels      []string `json:"labels,omitempty"`
	// AutoStart, when true, creates the task directly in 'start' rather
	// than 'pending'. Omitted = pending (the server default).
	AutoStart *bool `json:"auto_start,omitempty"`
}

func (c *Client) CreateTask(projectIDOrName string, in CreateTaskInput) (*Task, error) {
	var out Task
	path := fmt.Sprintf("/api/v1/projects/%s/tasks", url.PathEscape(projectIDOrName))
	if err := c.do("POST", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetTaskContext(taskID string) (*ContextSnapshot, error) {
	var out ContextSnapshot
	if err := c.do("GET", "/api/v1/tasks/"+url.PathEscape(taskID)+"/context", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RecallTaskMemory(taskID, query string) (*MemoryRecall, error) {
	var out MemoryRecall
	values := url.Values{"q": []string{query}}
	path := "/api/v1/tasks/" + url.PathEscape(taskID) + "/recall?" + values.Encode()
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CaptureLearningNote(
	projectIDOrName string,
	in CaptureLearningNoteInput,
) (*LearningNote, error) {
	var out LearningNote
	path := fmt.Sprintf("/api/v1/projects/%s/learning-notes", url.PathEscape(projectIDOrName))
	if err := c.do("POST", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListLearningNotes(
	projectIDOrName string,
	taskID string,
	status string,
	limit int,
) ([]LearningNote, error) {
	var path string
	if taskID != "" {
		path = "/api/v1/tasks/" + url.PathEscape(taskID) + "/learning-notes"
	} else {
		path = fmt.Sprintf("/api/v1/projects/%s/learning-notes", url.PathEscape(projectIDOrName))
	}
	values := url.Values{}
	if status != "" {
		values.Set("status", status)
	}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out struct {
		LearningNotes []LearningNote `json:"learning_notes"`
	}
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out.LearningNotes, nil
}

func (c *Client) PromoteLearningNote(id, evidence string, memoryClasses ...string) (*LearningNote, error) {
	var out LearningNote
	path := "/api/v1/learning-notes/" + url.PathEscape(id) + "/promote"
	memoryClass := "experience"
	if len(memoryClasses) > 0 && memoryClasses[0] != "" {
		memoryClass = memoryClasses[0]
	}
	if err := c.do("POST", path, map[string]string{
		"evidence": evidence, "memory_class": memoryClass,
	}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RejectLearningNote(id, reason string) (*LearningNote, error) {
	var out LearningNote
	path := "/api/v1/learning-notes/" + url.PathEscape(id) + "/reject"
	if err := c.do("POST", path, map[string]string{"reason": reason}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateLearningNote(id string, in UpdateLearningNoteInput) (*LearningNote, error) {
	var out LearningNote
	path := "/api/v1/learning-notes/" + url.PathEscape(id)
	if err := c.do("PATCH", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateCapsule(projectIDOrName string, in CreateCapsuleInput) (*ExplorationCapsule, error) {
	var out ExplorationCapsule
	path := fmt.Sprintf("/api/v1/projects/%s/capsules", url.PathEscape(projectIDOrName))
	if err := c.do("POST", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListCapsules(projectIDOrName, query, status string) ([]ExplorationCapsule, error) {
	values := url.Values{}
	if query != "" {
		values.Set("q", query)
	}
	if status != "" {
		values.Set("status", status)
	}
	path := fmt.Sprintf("/api/v1/projects/%s/capsules", url.PathEscape(projectIDOrName))
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out struct {
		Capsules []ExplorationCapsule `json:"capsules"`
	}
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out.Capsules, nil
}

func (c *Client) UpdateCapsuleStatus(id, status string) (*ExplorationCapsule, error) {
	var out ExplorationCapsule
	if err := c.do("PATCH", "/api/v1/capsules/"+url.PathEscape(id), map[string]string{"status": status}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateCapsule(id string, in UpdateCapsuleInput) (*ExplorationCapsule, error) {
	var out ExplorationCapsule
	if err := c.do("PATCH", "/api/v1/capsules/"+url.PathEscape(id), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RecordCapsuleUsage(id string, input RecordCapsuleUsageInput) (*MemoryImpact, error) {
	var out MemoryImpact
	if err := c.do("POST", "/api/v1/capsules/"+url.PathEscape(id)+"/usages", input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateMemoryRelation(id string, in CreateMemoryRelationInput) (*MemoryRelation, error) {
	var out MemoryRelation
	path := "/api/v1/capsules/" + url.PathEscape(id) + "/relations"
	if err := c.do("POST", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteMemoryRelation(id string) error {
	return c.do("DELETE", "/api/v1/memory-relations/"+url.PathEscape(id), nil, nil)
}

func (c *Client) GetLearningMetrics(projectIDOrName string) (*LearningMetrics, error) {
	var out LearningMetrics
	path := fmt.Sprintf("/api/v1/projects/%s/learning-metrics", url.PathEscape(projectIDOrName))
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type listTaskEventsResp struct {
	Events []TaskEvent `json:"events"`
}

func (c *Client) ListTaskEvents(taskID string) ([]TaskEvent, error) {
	var out listTaskEventsResp
	path := fmt.Sprintf("/api/v1/tasks/%s/events", url.PathEscape(taskID))
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out.Events, nil
}

type listTasksResp struct {
	Tasks []Task `json:"tasks"`
}

type ListTaskOptions struct {
	Owner     string
	Unclaimed bool
	Labels    []string
}

func (c *Client) ListTasks(projectIDOrName string, states []string, opts ...ListTaskOptions) ([]Task, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/tasks", url.PathEscape(projectIDOrName))
	q := url.Values{}
	if len(states) > 0 {
		q.Set("state", strings.Join(states, ","))
	}
	if len(opts) > 0 {
		if opts[0].Owner != "" {
			q.Set("owner", opts[0].Owner)
		}
		if opts[0].Unclaimed {
			q.Set("unclaimed", "true")
		}
		for _, label := range opts[0].Labels {
			q.Add("label", label)
		}
	}
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out listTasksResp
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out.Tasks, nil
}

func (c *Client) SearchTasks(projectIDOrName, query string, limit int) ([]Task, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/tasks/search", url.PathEscape(projectIDOrName))
	q := url.Values{}
	q.Set("q", query)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	path += "?" + q.Encode()
	var out listTasksResp
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out.Tasks, nil
}

type ListRunnableOptions struct {
	Labels []string
}

func (c *Client) ListRunnableTasks(projectIDOrName string, opts ...ListRunnableOptions) ([]Task, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/tasks/runnable", url.PathEscape(projectIDOrName))
	if len(opts) > 0 {
		q := url.Values{}
		for _, label := range opts[0].Labels {
			q.Add("label", label)
		}
		if encoded := q.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}
	var out listTasksResp
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out.Tasks, nil
}

type nextTaskResp struct {
	Task *Task `json:"task"`
}

type NextTaskOptions struct {
	Claim  bool
	Lease  string
	Labels []string
}

// NextRunnableTask returns the highest-priority runnable task or nil if none.
func (c *Client) NextRunnableTask(projectIDOrName string, opts ...NextTaskOptions) (*Task, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/tasks/next", url.PathEscape(projectIDOrName))
	if len(opts) > 0 {
		q := url.Values{}
		if opts[0].Claim {
			q.Set("claim", "true")
		}
		if opts[0].Lease != "" {
			q.Set("lease", opts[0].Lease)
		}
		for _, label := range opts[0].Labels {
			q.Add("label", label)
		}
		if encoded := q.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}
	var out nextTaskResp
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out.Task, nil
}

func (c *Client) GetTask(id string) (*Task, error) {
	var out Task
	path := fmt.Sprintf("/api/v1/tasks/%s", url.PathEscape(id))
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UpdateTaskInput struct {
	Title             *string   `json:"title,omitempty"`
	Description       *string   `json:"description,omitempty"`
	DescriptionAppend *string   `json:"description_append,omitempty"`
	Type              *string   `json:"type,omitempty"`
	State             *string   `json:"state,omitempty"`
	Priority          *int      `json:"priority,omitempty"`
	Labels            *[]string `json:"labels,omitempty"`
	LabelOps          *LabelOps `json:"label_ops,omitempty"`
	IfState           *string   `json:"if_state,omitempty"`
	Force             bool      `json:"force,omitempty"`
}

type LabelOps struct {
	Add    []string `json:"add,omitempty"`
	Remove []string `json:"remove,omitempty"`
}

func (c *Client) UpdateTask(id string, in UpdateTaskInput) (*Task, error) {
	var out Task
	path := fmt.Sprintf("/api/v1/tasks/%s", url.PathEscape(id))
	if err := c.do("PATCH", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteTask(id string) error {
	path := fmt.Sprintf("/api/v1/tasks/%s", url.PathEscape(id))
	return c.do("DELETE", path, nil, nil)
}

type ClaimTaskInput struct {
	Lease string `json:"lease,omitempty"`
}

func (c *Client) ClaimTask(id string, in ClaimTaskInput) (*Task, error) {
	var out Task
	path := fmt.Sprintf("/api/v1/tasks/%s/claim", url.PathEscape(id))
	if err := c.do("POST", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type HeartbeatTaskInput struct {
	Lease string `json:"lease,omitempty"`
}

func (c *Client) HeartbeatTask(id string, in HeartbeatTaskInput) (*Task, error) {
	var out Task
	path := fmt.Sprintf("/api/v1/tasks/%s/heartbeat", url.PathEscape(id))
	if err := c.do("POST", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type ReleaseTaskInput struct {
	Force bool `json:"force,omitempty"`
}

func (c *Client) ReleaseTask(id string, in ReleaseTaskInput) (*Task, error) {
	var out Task
	path := fmt.Sprintf("/api/v1/tasks/%s/release", url.PathEscape(id))
	if err := c.do("POST", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type addDepReq struct {
	DependsOn string `json:"depends_on"`
}

func (c *Client) AddDependency(taskID, dependsOnID string) error {
	path := fmt.Sprintf("/api/v1/tasks/%s/deps", url.PathEscape(taskID))
	return c.do("POST", path, addDepReq{DependsOn: dependsOnID}, nil)
}

func (c *Client) DeleteDependency(taskID, dependsOnID string) error {
	path := fmt.Sprintf(
		"/api/v1/tasks/%s/deps/%s",
		url.PathEscape(taskID),
		url.PathEscape(dependsOnID),
	)
	return c.do("DELETE", path, nil, nil)
}

type AddLinkInput struct {
	URL   string `json:"url"`
	Label string `json:"label"`
}

func (c *Client) AddLink(taskID string, in AddLinkInput) (*Link, error) {
	var out Link
	path := fmt.Sprintf("/api/v1/tasks/%s/links", url.PathEscape(taskID))
	if err := c.do("POST", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteLink(linkID string) error {
	path := fmt.Sprintf("/api/v1/links/%s", url.PathEscape(linkID))
	return c.do("DELETE", path, nil, nil)
}

type CreateDocInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func (c *Client) CreateDoc(taskID string, in CreateDocInput) (*Doc, error) {
	var out Doc
	path := fmt.Sprintf("/api/v1/tasks/%s/docs", url.PathEscape(taskID))
	if err := c.do("POST", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetDoc(docID string) (*Doc, error) {
	var out Doc
	path := fmt.Sprintf("/api/v1/docs/%s", url.PathEscape(docID))
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type UpdateDocInput struct {
	Title   *string `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
}

func (c *Client) UpdateDoc(docID string, in UpdateDocInput) (*Doc, error) {
	var out Doc
	path := fmt.Sprintf("/api/v1/docs/%s", url.PathEscape(docID))
	if err := c.do("PATCH", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteDoc(docID string) error {
	path := fmt.Sprintf("/api/v1/docs/%s", url.PathEscape(docID))
	return c.do("DELETE", path, nil, nil)
}

func (c *Client) UploadImage(taskID, filePath string) (*Image, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/api/v1/tasks/%s/images", c.BaseURL, url.PathEscape(taskID))
	req, err := http.NewRequest("POST", path, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Taskline-Client", "cli")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, decodeServerError(resp)
	}
	var out Image
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ─── plumbing ───────────────────────────────────────────────────────────

func (c *Client) do(method, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Taskline-Client", "cli")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return decodeServerError(resp)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type errResp struct {
	Error string `json:"error"`
}

func decodeServerError(resp *http.Response) error {
	raw, _ := io.ReadAll(resp.Body)
	var e errResp
	if json.Unmarshal(raw, &e) == nil && e.Error != "" {
		return fmt.Errorf("runengram %d: %s", resp.StatusCode, e.Error)
	}
	if msg := strings.TrimSpace(string(raw)); msg != "" {
		return fmt.Errorf("runengram %d: %s", resp.StatusCode, msg)
	}
	return fmt.Errorf("runengram %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
}
