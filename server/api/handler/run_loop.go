package handler

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"

	"taskline_server/api/model"
	"taskline_server/internal/service"
)

type startRunReq struct {
	AgentTool          string                    `json:"agent_tool"`
	WorkflowTemplate   string                    `json:"workflow_template"`
	WorkflowDefinition *model.WorkflowDefinition `json:"workflow_definition"`
}

type saveRunCheckpointReq struct {
	Status   string `json:"status"`
	Summary  string `json:"summary"`
	NextStep string `json:"next_step"`
}

type recordRunEventReq struct {
	Kind    string         `json:"kind"`
	Summary string         `json:"summary"`
	Details map[string]any `json:"details"`
}

type finishRunReq struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

func (h *Handler) startOrResumeRun(ctx context.Context, c *app.RequestContext) {
	agent, ok := h.requireAgent(ctx, c)
	if !ok {
		return
	}
	ctx = service.WithActor(ctx, agent.Name)
	var req startRunReq
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	run, resumed, err := h.svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: c.Param("id"), AgentName: agent.Name,
		AgentTool:          model.AgentTool(req.AgentTool),
		WorkflowTemplate:   model.WorkflowTemplate(req.WorkflowTemplate),
		WorkflowDefinition: req.WorkflowDefinition,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	status := http.StatusCreated
	if resumed {
		status = http.StatusOK
	}
	writeJSON(c, status, map[string]any{"run": run, "resumed": resumed})
}

func (h *Handler) listAgentRuns(ctx context.Context, c *app.RequestContext) {
	runs, err := h.svc.ListAgentRuns(ctx, c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, map[string]any{"runs": runs})
}

func (h *Handler) getAgentRun(ctx context.Context, c *app.RequestContext) {
	run, err := h.svc.GetAgentRun(ctx, c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, run)
}

func (h *Handler) saveRunCheckpoint(ctx context.Context, c *app.RequestContext) {
	agent, ok := h.requireAgent(ctx, c)
	if !ok {
		return
	}
	ctx = service.WithActor(ctx, agent.Name)
	var req saveRunCheckpointReq
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	run, err := h.svc.SaveRunCheckpoint(ctx, service.SaveRunCheckpointInput{
		RunID: c.Param("id"), AgentName: agent.Name,
		Status: model.RunStatus(req.Status), Summary: req.Summary, NextStep: req.NextStep,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, run)
}

func (h *Handler) recordRunEvent(ctx context.Context, c *app.RequestContext) {
	agent, ok := h.requireAgent(ctx, c)
	if !ok {
		return
	}
	ctx = service.WithActor(ctx, agent.Name)
	var req recordRunEventReq
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	event, err := h.svc.RecordRunEvent(ctx, service.RecordRunEventInput{
		RunID: c.Param("id"), AgentName: agent.Name,
		Kind: model.RunEventKind(req.Kind), Summary: req.Summary, Details: req.Details,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusCreated, event)
}

func (h *Handler) finishRun(ctx context.Context, c *app.RequestContext) {
	agent, ok := h.requireAgent(ctx, c)
	if !ok {
		return
	}
	ctx = service.WithActor(ctx, agent.Name)
	var req finishRunReq
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	run, err := h.svc.FinishRun(ctx, service.FinishRunInput{
		RunID: c.Param("id"), AgentName: agent.Name,
		Status: model.RunStatus(req.Status), Summary: req.Summary,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, run)
}

func (h *Handler) taskResumeContext(ctx context.Context, c *app.RequestContext) {
	agent, ok := h.optionalAgent(ctx, c)
	if !ok {
		return
	}
	agentName := ""
	if agent != nil {
		agentName = agent.Name
	}
	resume, err := h.svc.GetTaskResumeContext(ctx, c.Param("id"), agentName)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, resume)
}
