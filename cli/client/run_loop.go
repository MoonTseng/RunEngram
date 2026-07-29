package client

import (
	"fmt"
	"net/url"
)

type AgentRun struct {
	ID               string `json:"id"`
	TaskID           string `json:"task_id"`
	ProjectID        string `json:"project_id"`
	AgentName        string `json:"agent_name"`
	AgentTool        string `json:"agent_tool"`
	WorkflowTemplate string `json:"workflow_template"`
	WorkflowVersion  int    `json:"workflow_version"`
	Status           string `json:"status"`
	Summary          string `json:"summary"`
	NextStep         string `json:"next_step"`
	StartedAt        int64  `json:"started_at"`
	UpdatedAt        int64  `json:"updated_at"`
	CompletedAt      int64  `json:"completed_at"`
}

type RunEvent struct {
	ID        string         `json:"id"`
	RunID     string         `json:"run_id"`
	TaskID    string         `json:"task_id"`
	Actor     string         `json:"actor"`
	Kind      string         `json:"kind"`
	Summary   string         `json:"summary"`
	Details   map[string]any `json:"details"`
	CreatedAt int64          `json:"created_at"`
}

type RunStartResult struct {
	Run     AgentRun `json:"run"`
	Resumed bool     `json:"resumed"`
}

type WorkflowNodeSpec struct {
	Key        string   `json:"key"`
	Title      string   `json:"title"`
	Capability string   `json:"capability"`
	Kind       string   `json:"kind"`
	DependsOn  []string `json:"depends_on"`
}

type WorkflowDefinition struct {
	Template string             `json:"template"`
	Version  int                `json:"version"`
	Nodes    []WorkflowNodeSpec `json:"nodes"`
}

type TaskResumeContext struct {
	Snapshot     ContextSnapshot `json:"snapshot"`
	LatestRun    *AgentRun       `json:"latest_run,omitempty"`
	RecentEvents []TaskEvent     `json:"recent_events"`
	WorkGraph    *RunWorkGraph   `json:"work_graph,omitempty"`
}

func (c *Client) StartAgentRun(taskID, agentTool string) (*RunStartResult, error) {
	return c.StartAgentRunWithWorkflow(taskID, agentTool, "")
}

func (c *Client) StartAgentRunWithWorkflow(
	taskID, agentTool, workflow string,
) (*RunStartResult, error) {
	return c.StartAgentRunWithDefinition(taskID, agentTool, workflow, nil)
}

func (c *Client) StartAgentRunWithDefinition(
	taskID, agentTool, workflow string,
	definition *WorkflowDefinition,
) (*RunStartResult, error) {
	var out RunStartResult
	path := fmt.Sprintf("/api/v1/tasks/%s/runs", url.PathEscape(taskID))
	input := map[string]any{"agent_tool": agentTool}
	if workflow != "" {
		input["workflow_template"] = workflow
	}
	if definition != nil {
		input["workflow_definition"] = definition
	}
	if err := c.do("POST", path, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type RunNode struct {
	ID               string   `json:"id"`
	RunID            string   `json:"run_id"`
	Key              string   `json:"key"`
	Title            string   `json:"title"`
	Capability       string   `json:"capability"`
	Kind             string   `json:"kind"`
	Position         int      `json:"position"`
	DependsOn        []string `json:"depends_on"`
	Status           string   `json:"status"`
	Attempt          int      `json:"attempt"`
	Summary          string   `json:"summary"`
	NextStep         string   `json:"next_step"`
	ArtifactIDs      []string `json:"artifact_ids"`
	Evidence         string   `json:"evidence"`
	InputFingerprint string   `json:"input_fingerprint"`
	StartedAt        int64    `json:"started_at"`
	CompletedAt      int64    `json:"completed_at"`
	UpdatedAt        int64    `json:"updated_at"`
}

type RunInterrupt struct {
	ID          string   `json:"id"`
	RunID       string   `json:"run_id"`
	NodeKey     string   `json:"node_key"`
	Kind        string   `json:"kind"`
	Prompt      string   `json:"prompt"`
	Options     []string `json:"options"`
	Status      string   `json:"status"`
	Response    string   `json:"response"`
	RequestedBy string   `json:"requested_by"`
	RespondedBy string   `json:"responded_by"`
	CreatedAt   int64    `json:"created_at"`
	ResolvedAt  int64    `json:"resolved_at"`
}

type RunWorkGraph struct {
	RunID              string         `json:"run_id"`
	Template           string         `json:"template"`
	Version            int            `json:"version"`
	Nodes              []RunNode      `json:"nodes"`
	Interrupts         []RunInterrupt `json:"interrupts"`
	CompletedNodeCount int            `json:"completed_node_count"`
	VerifiedNodeCount  int            `json:"verified_node_count"`
	ArtifactCount      int            `json:"artifact_count"`
	OpenInterruptCount int            `json:"open_interrupt_count"`
	ProgressPercent    int            `json:"progress_percent"`
}

type UpdateRunNodeInput struct {
	Status           string
	Summary          string
	NextStep         string
	ArtifactIDs      []string
	Evidence         string
	InputFingerprint string
}

func (c *Client) GetRunWorkGraph(runID string) (*RunWorkGraph, error) {
	var out RunWorkGraph
	path := fmt.Sprintf("/api/v1/runs/%s/work-graph", url.PathEscape(runID))
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateRunNode(
	runID, nodeKey string,
	input UpdateRunNodeInput,
) (*RunNode, error) {
	var out RunNode
	path := fmt.Sprintf(
		"/api/v1/runs/%s/nodes/%s",
		url.PathEscape(runID),
		url.PathEscape(nodeKey),
	)
	body := map[string]any{
		"status": input.Status, "summary": input.Summary,
		"next_step": input.NextStep, "artifact_ids": input.ArtifactIDs,
		"evidence": input.Evidence, "input_fingerprint": input.InputFingerprint,
	}
	if err := c.do("PATCH", path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateRunInterrupt(
	runID, nodeKey, kind, prompt string,
	options []string,
) (*RunInterrupt, error) {
	var out RunInterrupt
	path := fmt.Sprintf("/api/v1/runs/%s/interrupts", url.PathEscape(runID))
	body := map[string]any{
		"node_key": nodeKey, "kind": kind, "prompt": prompt, "options": options,
	}
	if err := c.do("POST", path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ResolveRunInterrupt(
	interruptID, response string,
	reject bool,
) (*RunInterrupt, error) {
	var out RunInterrupt
	path := fmt.Sprintf("/api/v1/interrupts/%s", url.PathEscape(interruptID))
	body := map[string]any{"response": response, "reject": reject}
	if err := c.do("PATCH", path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetAgentRun(runID string) (*AgentRun, error) {
	var out AgentRun
	path := fmt.Sprintf("/api/v1/runs/%s", url.PathEscape(runID))
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListAgentRuns(taskID string) ([]AgentRun, error) {
	var out struct {
		Runs []AgentRun `json:"runs"`
	}
	path := fmt.Sprintf("/api/v1/tasks/%s/runs", url.PathEscape(taskID))
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out.Runs, nil
}

func (c *Client) SaveRunCheckpoint(
	runID, status, summary, nextStep string,
) (*AgentRun, error) {
	var out AgentRun
	path := fmt.Sprintf("/api/v1/runs/%s/checkpoint", url.PathEscape(runID))
	in := map[string]any{
		"status": status, "summary": summary, "next_step": nextStep,
	}
	if err := c.do("PATCH", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RecordRunEvent(
	runID, kind, summary string,
	details map[string]any,
) (*RunEvent, error) {
	var out RunEvent
	path := fmt.Sprintf("/api/v1/runs/%s/events", url.PathEscape(runID))
	in := map[string]any{"kind": kind, "summary": summary, "details": details}
	if err := c.do("POST", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) FinishAgentRun(
	runID, status, summary string,
) (*AgentRun, error) {
	var out AgentRun
	path := fmt.Sprintf("/api/v1/runs/%s/finish", url.PathEscape(runID))
	in := map[string]any{"status": status, "summary": summary}
	if err := c.do("POST", path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetTaskResumeContext(taskID string) (*TaskResumeContext, error) {
	var out TaskResumeContext
	path := fmt.Sprintf("/api/v1/tasks/%s/resume", url.PathEscape(taskID))
	if err := c.do("GET", path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
