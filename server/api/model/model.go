package model

import "errors"

// TaskType is the kind of work a task represents.
type TaskType string

const (
	TaskTypeFeature TaskType = "feature"
	TaskTypeBug     TaskType = "bug"
	TaskTypeDocs    TaskType = "docs"
)

func (t TaskType) Valid() bool {
	switch t {
	case TaskTypeFeature, TaskTypeBug, TaskTypeDocs:
		return true
	}
	return false
}

// TaskState is the current position in the workflow.
type TaskState string

const (
	// StatePending is a parking lot — created but explicitly not yet
	// runnable. ListRunnableTasks skips it.
	StatePending TaskState = "pending"
	StateStart   TaskState = "start"
	StateSpec    TaskState = "spec"
	StateDev     TaskState = "dev"
	StateTest    TaskState = "test"
	StateReview  TaskState = "review"
	StateDone    TaskState = "done"
)

// stateOrder reflects the canonical workflow position. Transitions are
// validated for state membership only — movement in either direction is
// allowed, since work sometimes legitimately needs to drop back (e.g. a
// review surfaces a bug that must return to dev). 'pending' lives off to
// the side of the main pipeline; any state may drop into it.
var stateOrder = map[TaskState]int{
	StatePending: -1,
	StateStart:   0,
	StateSpec:    1,
	StateDev:     2,
	StateTest:    3,
	StateReview:  4,
	StateDone:    5,
}

func (s TaskState) Valid() bool {
	_, ok := stateOrder[s]
	return ok
}

// CanTransitionTo returns nil if moving from s to next is allowed.
// Backward moves are permitted; only invalid state names are rejected.
func (s TaskState) CanTransitionTo(next TaskState) error {
	if !s.Valid() {
		return errors.New("invalid current state")
	}
	if !next.Valid() {
		return errors.New("invalid next state")
	}
	return nil
}

// IsTerminal reports whether the task is in its final state.
func (s TaskState) IsTerminal() bool { return s == StateDone }

// Project is a workspace that owns tasks.
type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// Agent is a local worker identity used to derive task claim ownership.
type Agent struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// ActiveClaim is the compact task shape exposed by the authenticated status
// endpoint. It intentionally omits task details and attachments.
type ActiveClaim struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	ClaimedAt      int64  `json:"claimed_at"`
	ClaimedForMS   int64  `json:"claimed_for_ms"`
	LeaseExpiresAt int64  `json:"lease_expires_at"`
}

// ServerStatus proves server reachability and, when authenticated, agent
// identity plus currently live claims.
type ServerStatus struct {
	OK          bool          `json:"ok"`
	ServerTime  int64         `json:"server_time"`
	Agent       *Agent        `json:"agent,omitempty"`
	ActiveTasks []ActiveClaim `json:"active_tasks"`
}

// Task is the unit of work tracked under a project.
type Task struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"project_id"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Type           TaskType       `json:"type"`
	State          TaskState      `json:"state"`
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

// TaskEvent is one append-only mutation record for a task. Details stays
// structured so clients can render full before/after values without parsing
// human-readable summaries.
type TaskEvent struct {
	ID        string         `json:"id"`
	TaskID    string         `json:"task_id"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Summary   string         `json:"summary"`
	Details   map[string]any `json:"details"`
	CreatedAt int64          `json:"created_at"`
}

type AgentTool string

const (
	AgentToolCodex      AgentTool = "codex"
	AgentToolClaudeCode AgentTool = "claude-code"
	AgentToolPi         AgentTool = "pi"
	AgentToolOther      AgentTool = "other"
)

func (t AgentTool) Valid() bool {
	switch t {
	case AgentToolCodex, AgentToolClaudeCode, AgentToolPi, AgentToolOther:
		return true
	default:
		return false
	}
}

type RunStatus string

const (
	RunStatusRunning   RunStatus = "running"
	RunStatusBlocked   RunStatus = "blocked"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)

func (s RunStatus) Valid() bool {
	switch s {
	case RunStatusRunning, RunStatusBlocked, RunStatusCompleted, RunStatusFailed:
		return true
	default:
		return false
	}
}

func (s RunStatus) Terminal() bool {
	return s == RunStatusCompleted || s == RunStatusFailed
}

type RunEventKind string

const (
	RunEventStarted            RunEventKind = "run.started"
	RunEventResumed            RunEventKind = "run.resumed"
	RunEventNodeUpdated        RunEventKind = "workflow.node.updated"
	RunEventInterruptCreated   RunEventKind = "workflow.interrupt.created"
	RunEventInterruptResolved  RunEventKind = "workflow.interrupt.resolved"
	RunEventToolCalled         RunEventKind = "tool.called"
	RunEventCheckpointSaved    RunEventKind = "checkpoint.saved"
	RunEventBlocked            RunEventKind = "run.blocked"
	RunEventVerificationPassed RunEventKind = "verification.passed"
	RunEventLearningDiscovered RunEventKind = "learning.discovered"
	RunEventCompleted          RunEventKind = "run.completed"
	RunEventFailed             RunEventKind = "run.failed"
)

func (k RunEventKind) Valid() bool {
	switch k {
	case RunEventStarted, RunEventResumed, RunEventToolCalled,
		RunEventCheckpointSaved, RunEventBlocked, RunEventVerificationPassed,
		RunEventLearningDiscovered, RunEventCompleted, RunEventFailed,
		RunEventNodeUpdated, RunEventInterruptCreated, RunEventInterruptResolved:
		return true
	default:
		return false
	}
}

type AgentRun struct {
	ID               string           `json:"id"`
	TaskID           string           `json:"task_id"`
	ProjectID        string           `json:"project_id"`
	AgentName        string           `json:"agent_name"`
	AgentTool        AgentTool        `json:"agent_tool"`
	Status           RunStatus        `json:"status"`
	WorkflowTemplate WorkflowTemplate `json:"workflow_template"`
	WorkflowVersion  int              `json:"workflow_version"`
	Summary          string           `json:"summary"`
	NextStep         string           `json:"next_step"`
	StartedAt        int64            `json:"started_at"`
	UpdatedAt        int64            `json:"updated_at"`
	CompletedAt      int64            `json:"completed_at"`
}

type WorkflowTemplate string

const (
	WorkflowTemplateSingleLoop      WorkflowTemplate = "single-loop"
	WorkflowTemplateEngineeringFlow WorkflowTemplate = "engineering-flow"
)

func (t WorkflowTemplate) Valid() bool {
	value := string(t)
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for index, char := range value {
		valid := char >= 'a' && char <= 'z' ||
			char >= '0' && char <= '9' ||
			char == '-'
		if !valid || index == 0 && (char < 'a' || char > 'z') {
			return false
		}
	}
	return value[len(value)-1] != '-'
}

// WorkflowNodeSpec is the portable contract between an external workflow
// adapter and RunEngram's durable Work Graph.
type WorkflowNodeSpec struct {
	Key        string   `json:"key"`
	Title      string   `json:"title"`
	Capability string   `json:"capability"`
	Kind       string   `json:"kind"`
	DependsOn  []string `json:"depends_on"`
}

// WorkflowDefinition lets any SOP describe its durable outer graph without
// moving the SOP's inner Agent behavior into RunEngram.
type WorkflowDefinition struct {
	Template WorkflowTemplate   `json:"template"`
	Version  int                `json:"version"`
	Nodes    []WorkflowNodeSpec `json:"nodes"`
}

type RunNodeStatus string

const (
	RunNodePending   RunNodeStatus = "pending"
	RunNodeReady     RunNodeStatus = "ready"
	RunNodeRunning   RunNodeStatus = "running"
	RunNodeWaiting   RunNodeStatus = "waiting"
	RunNodeCompleted RunNodeStatus = "completed"
	RunNodeFailed    RunNodeStatus = "failed"
	RunNodeSkipped   RunNodeStatus = "skipped"
)

func (s RunNodeStatus) Valid() bool {
	switch s {
	case RunNodePending, RunNodeReady, RunNodeRunning, RunNodeWaiting,
		RunNodeCompleted, RunNodeFailed, RunNodeSkipped:
		return true
	default:
		return false
	}
}

func (s RunNodeStatus) SatisfiesDependency() bool {
	return s == RunNodeCompleted || s == RunNodeSkipped
}

type RunNode struct {
	ID               string        `json:"id"`
	RunID            string        `json:"run_id"`
	Key              string        `json:"key"`
	Title            string        `json:"title"`
	Capability       string        `json:"capability"`
	Kind             string        `json:"kind"`
	Position         int           `json:"position"`
	DependsOn        []string      `json:"depends_on"`
	Status           RunNodeStatus `json:"status"`
	Attempt          int           `json:"attempt"`
	Summary          string        `json:"summary"`
	NextStep         string        `json:"next_step"`
	ArtifactIDs      []string      `json:"artifact_ids"`
	Evidence         string        `json:"evidence"`
	InputFingerprint string        `json:"input_fingerprint"`
	StartedAt        int64         `json:"started_at"`
	CompletedAt      int64         `json:"completed_at"`
	UpdatedAt        int64         `json:"updated_at"`
}

type RunInterruptKind string

const (
	RunInterruptApproval RunInterruptKind = "approval"
	RunInterruptQuestion RunInterruptKind = "question"
	RunInterruptChoice   RunInterruptKind = "choice"
	RunInterruptConflict RunInterruptKind = "conflict"
)

func (k RunInterruptKind) Valid() bool {
	switch k {
	case RunInterruptApproval, RunInterruptQuestion, RunInterruptChoice,
		RunInterruptConflict:
		return true
	default:
		return false
	}
}

type RunInterruptStatus string

const (
	RunInterruptPending  RunInterruptStatus = "pending"
	RunInterruptAnswered RunInterruptStatus = "answered"
	RunInterruptRejected RunInterruptStatus = "rejected"
)

type RunInterrupt struct {
	ID          string             `json:"id"`
	RunID       string             `json:"run_id"`
	NodeKey     string             `json:"node_key"`
	Kind        RunInterruptKind   `json:"kind"`
	Prompt      string             `json:"prompt"`
	Options     []string           `json:"options"`
	Status      RunInterruptStatus `json:"status"`
	Response    string             `json:"response"`
	RequestedBy string             `json:"requested_by"`
	RespondedBy string             `json:"responded_by"`
	CreatedAt   int64              `json:"created_at"`
	ResolvedAt  int64              `json:"resolved_at"`
}

type RunWorkGraph struct {
	RunID              string           `json:"run_id"`
	Template           WorkflowTemplate `json:"template"`
	Version            int              `json:"version"`
	Nodes              []*RunNode       `json:"nodes"`
	Interrupts         []*RunInterrupt  `json:"interrupts"`
	CompletedNodeCount int              `json:"completed_node_count"`
	VerifiedNodeCount  int              `json:"verified_node_count"`
	ArtifactCount      int              `json:"artifact_count"`
	OpenInterruptCount int              `json:"open_interrupt_count"`
	ProgressPercent    int              `json:"progress_percent"`
}

type RunEvent struct {
	ID        string         `json:"id"`
	RunID     string         `json:"run_id"`
	TaskID    string         `json:"task_id"`
	Actor     string         `json:"actor"`
	Kind      RunEventKind   `json:"kind"`
	Summary   string         `json:"summary"`
	Details   map[string]any `json:"details"`
	CreatedAt int64          `json:"created_at"`
}

type TaskResumeContext struct {
	Snapshot     ContextSnapshot `json:"snapshot"`
	LatestRun    *AgentRun       `json:"latest_run,omitempty"`
	WorkGraph    *RunWorkGraph   `json:"work_graph,omitempty"`
	RecentEvents []*TaskEvent    `json:"recent_events"`
}

type CapsuleStatus string

const (
	CapsuleStatusActive   CapsuleStatus = "active"
	CapsuleStatusStale    CapsuleStatus = "stale"
	CapsuleStatusArchived CapsuleStatus = "archived"
)

func (s CapsuleStatus) Valid() bool {
	return s == CapsuleStatusActive || s == CapsuleStatusStale || s == CapsuleStatusArchived
}

type MemoryClass string

const (
	MemoryClassExperience  MemoryClass = "experience"
	MemoryClassProjectRule MemoryClass = "project-rule"
)

func (c MemoryClass) Valid() bool {
	return c == MemoryClassExperience || c == MemoryClassProjectRule
}

type MemoryRelationType string

const (
	MemoryRelationDerivedFrom   MemoryRelationType = "derived-from"
	MemoryRelationValidatedBy   MemoryRelationType = "validated-by"
	MemoryRelationAppliesTo     MemoryRelationType = "applies-to"
	MemoryRelationSupersedes    MemoryRelationType = "supersedes"
	MemoryRelationConflictsWith MemoryRelationType = "conflicts-with"
	MemoryRelationCausedBy      MemoryRelationType = "caused-by"
)

func (r MemoryRelationType) Valid() bool {
	return r == MemoryRelationDerivedFrom || r == MemoryRelationValidatedBy ||
		r == MemoryRelationAppliesTo || r == MemoryRelationSupersedes ||
		r == MemoryRelationConflictsWith || r == MemoryRelationCausedBy
}

type MemoryRelationTargetKind string

const (
	MemoryRelationTargetCapsule  MemoryRelationTargetKind = "capsule"
	MemoryRelationTargetTask     MemoryRelationTargetKind = "task"
	MemoryRelationTargetArtifact MemoryRelationTargetKind = "artifact"
	MemoryRelationTargetScope    MemoryRelationTargetKind = "scope"
)

func (k MemoryRelationTargetKind) Valid() bool {
	return k == MemoryRelationTargetCapsule || k == MemoryRelationTargetTask ||
		k == MemoryRelationTargetArtifact || k == MemoryRelationTargetScope
}

type MemoryRelationDirection string

const (
	MemoryRelationOutgoing MemoryRelationDirection = "outgoing"
	MemoryRelationIncoming MemoryRelationDirection = "incoming"
)

type MemoryValidation string

const (
	MemoryValidationVerified MemoryValidation = "verified"
	MemoryValidationTrusted  MemoryValidation = "trusted"
	MemoryValidationDisputed MemoryValidation = "disputed"
	MemoryValidationStale    MemoryValidation = "stale"
)

type CapsuleOutcome string

const (
	CapsuleOutcomeUsed     CapsuleOutcome = "used"
	CapsuleOutcomeHelpful  CapsuleOutcome = "helpful"
	CapsuleOutcomeRejected CapsuleOutcome = "rejected"
	CapsuleOutcomeStale    CapsuleOutcome = "stale"
)

func (o CapsuleOutcome) Valid() bool {
	return o == CapsuleOutcomeUsed || o == CapsuleOutcomeHelpful ||
		o == CapsuleOutcomeRejected || o == CapsuleOutcomeStale
}

type LearningNoteKind string

const (
	LearningNoteHumanCorrection LearningNoteKind = "human-correction"
	LearningNoteAgentRecovery   LearningNoteKind = "agent-recovery"
)

func (k LearningNoteKind) Valid() bool {
	return k == LearningNoteHumanCorrection || k == LearningNoteAgentRecovery
}

type LearningNoteStatus string

const (
	LearningNotePending  LearningNoteStatus = "pending"
	LearningNotePromoted LearningNoteStatus = "promoted"
	LearningNoteRejected LearningNoteStatus = "rejected"
)

func (s LearningNoteStatus) Valid() bool {
	return s == LearningNotePending || s == LearningNotePromoted || s == LearningNoteRejected
}

// LearningNote is an untrusted learning candidate captured from one agent run.
// Only promoted notes become reusable Exploration Capsules.
type LearningNote struct {
	ID              string             `json:"id"`
	ProjectID       string             `json:"project_id"`
	SourceTaskID    string             `json:"source_task_id"`
	Kind            LearningNoteKind   `json:"kind"`
	Trigger         string             `json:"trigger"`
	Guidance        string             `json:"guidance"`
	Scope           string             `json:"scope"`
	Labels          []string           `json:"labels"`
	Fingerprints    []string           `json:"fingerprints"`
	Producer        string             `json:"producer"`
	Status          LearningNoteStatus `json:"status"`
	Evidence        string             `json:"evidence"`
	CapsuleID       string             `json:"capsule_id"`
	RejectionReason string             `json:"rejection_reason"`
	CreatedAt       int64              `json:"created_at"`
	UpdatedAt       int64              `json:"updated_at"`
	ResolvedAt      int64              `json:"resolved_at"`
}

// ExplorationCapsule is verified, reusable engineering knowledge.
type ExplorationCapsule struct {
	ID            string           `json:"id"`
	ProjectID     string           `json:"project_id"`
	SourceTaskID  string           `json:"source_task_id"`
	MemoryClass   MemoryClass      `json:"memory_class"`
	Trigger       string           `json:"trigger"`
	Title         string           `json:"title"`
	Summary       string           `json:"summary"`
	Scope         string           `json:"scope"`
	Evidence      string           `json:"evidence"`
	Labels        []string         `json:"labels"`
	Fingerprints  []string         `json:"fingerprints"`
	Producer      string           `json:"producer"`
	Status        CapsuleStatus    `json:"status"`
	Validation    MemoryValidation `json:"validation"`
	Confidence    float64          `json:"confidence"`
	UseCount      int              `json:"use_count"`
	HelpfulCount  int              `json:"helpful_count"`
	RejectedCount int              `json:"rejected_count"`
	Relations     []MemoryRelation `json:"relations"`
	CreatedAt     int64            `json:"created_at"`
	UpdatedAt     int64            `json:"updated_at"`
}

// MemoryRelation connects verified memory to evidence, scope, or other memory.
type MemoryRelation struct {
	ID              string                   `json:"id"`
	ProjectID       string                   `json:"project_id"`
	SourceCapsuleID string                   `json:"source_capsule_id"`
	Type            MemoryRelationType       `json:"type"`
	TargetKind      MemoryRelationTargetKind `json:"target_kind"`
	TargetRef       string                   `json:"target_ref"`
	Note            string                   `json:"note"`
	Direction       MemoryRelationDirection  `json:"direction"`
	CreatedAt       int64                    `json:"created_at"`
}

// MemoryRecallExplanation makes automatic context selection inspectable.
type MemoryRecallReason struct {
	Code  string `json:"code"`
	Value string `json:"value,omitempty"`
}

type MemoryRecallExplanation struct {
	CapsuleID string               `json:"capsule_id"`
	Score     float64              `json:"score"`
	Reasons   []MemoryRecallReason `json:"reasons"`
	Warnings  []string             `json:"warnings"`
}

// ContextSnapshot freezes task input and suggested memory at first read.
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

// MemoryRecall is a live, query-specific recall during task execution.
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
	ID        string         `json:"id"`
	CapsuleID string         `json:"capsule_id"`
	TaskID    string         `json:"task_id"`
	Outcome   CapsuleOutcome `json:"outcome"`
	Notes     string         `json:"notes"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
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

// Link is a URL attached to a task — typically a spec doc, PR, technical
// note, or other artifact the agent wants to keep alongside the task.
type Link struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	URL       string `json:"url"`
	Label     string `json:"label"`
	CreatedAt int64  `json:"created_at"`
}

// Doc is a Markdown document attached to a task. Task list/detail responses
// include metadata and URL; the content field is populated only by doc-specific
// endpoints.
type Doc struct {
	ID          string `json:"id"`
	TaskID      string `json:"task_id"`
	Title       string `json:"title"`
	URL         string `json:"url,omitempty"`
	Content     string `json:"content,omitempty"`
	StoragePath string `json:"-"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// Image is a binary attachment uploaded against a task.
type Image struct {
	ID          string `json:"id"`
	TaskID      string `json:"task_id"`
	Filename    string `json:"filename"`
	MimeType    string `json:"mime_type"`
	SizeBytes   int64  `json:"size_bytes"`
	URL         string `json:"url,omitempty"`
	StoragePath string `json:"-"`
	UploadedAt  int64  `json:"uploaded_at"`
}
