package client

import (
	"fmt"
	"net/url"
)

type AgentRun struct {
	ID          string `json:"id"`
	TaskID      string `json:"task_id"`
	ProjectID   string `json:"project_id"`
	AgentName   string `json:"agent_name"`
	AgentTool   string `json:"agent_tool"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	NextStep    string `json:"next_step"`
	StartedAt   int64  `json:"started_at"`
	UpdatedAt   int64  `json:"updated_at"`
	CompletedAt int64  `json:"completed_at"`
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

type TaskResumeContext struct {
	Snapshot     ContextSnapshot `json:"snapshot"`
	LatestRun    *AgentRun       `json:"latest_run,omitempty"`
	RecentEvents []TaskEvent     `json:"recent_events"`
}

func (c *Client) StartAgentRun(taskID, agentTool string) (*RunStartResult, error) {
	var out RunStartResult
	path := fmt.Sprintf("/api/v1/tasks/%s/runs", url.PathEscape(taskID))
	if err := c.do("POST", path, map[string]any{"agent_tool": agentTool}, &out); err != nil {
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
