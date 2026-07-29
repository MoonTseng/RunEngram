package handler

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"

	"taskline_server/api/model"
	"taskline_server/internal/service"
)

type updateRunNodeReq struct {
	Status           string   `json:"status"`
	Summary          string   `json:"summary"`
	NextStep         string   `json:"next_step"`
	ArtifactIDs      []string `json:"artifact_ids"`
	Evidence         string   `json:"evidence"`
	InputFingerprint string   `json:"input_fingerprint"`
}

type createRunInterruptReq struct {
	NodeKey string   `json:"node_key"`
	Kind    string   `json:"kind"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
}

type resolveRunInterruptReq struct {
	Response string `json:"response"`
	Reject   bool   `json:"reject"`
}

func (h *Handler) getRunWorkGraph(ctx context.Context, c *app.RequestContext) {
	graph, err := h.svc.GetRunWorkGraph(ctx, c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, graph)
}

func (h *Handler) updateRunNode(ctx context.Context, c *app.RequestContext) {
	agent, ok := h.requireAgent(ctx, c)
	if !ok {
		return
	}
	ctx = service.WithActor(ctx, agent.Name)
	var req updateRunNodeReq
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	node, err := h.svc.UpdateRunNode(ctx, service.UpdateRunNodeInput{
		RunID: c.Param("id"), AgentName: agent.Name, NodeKey: c.Param("node"),
		Status: model.RunNodeStatus(req.Status), Summary: req.Summary,
		NextStep: req.NextStep, ArtifactIDs: req.ArtifactIDs,
		Evidence: req.Evidence, InputFingerprint: req.InputFingerprint,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, node)
}

func (h *Handler) createRunInterrupt(ctx context.Context, c *app.RequestContext) {
	agent, ok := h.requireAgent(ctx, c)
	if !ok {
		return
	}
	ctx = service.WithActor(ctx, agent.Name)
	var req createRunInterruptReq
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	interrupt, err := h.svc.CreateRunInterrupt(ctx, service.CreateRunInterruptInput{
		RunID: c.Param("id"), AgentName: agent.Name, NodeKey: req.NodeKey,
		Kind: model.RunInterruptKind(req.Kind), Prompt: req.Prompt,
		Options: req.Options,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusCreated, interrupt)
}

func (h *Handler) resolveRunInterrupt(ctx context.Context, c *app.RequestContext) {
	var ok bool
	ctx, ok = h.withRequestActor(ctx, c)
	if !ok {
		return
	}
	var req resolveRunInterruptReq
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	interrupt, err := h.svc.ResolveRunInterrupt(ctx, service.ResolveRunInterruptInput{
		InterruptID: c.Param("id"), Response: req.Response, Reject: req.Reject,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, interrupt)
}
