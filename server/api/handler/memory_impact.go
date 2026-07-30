package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"taskline_server/api/model"
	"taskline_server/internal/service"
	"taskline_server/internal/store"
)

type updateMemoryImpactReq struct {
	State             model.MemoryImpactState      `json:"state"`
	Stage             string                       `json:"stage"`
	Notes             string                       `json:"notes"`
	Evidence          []model.MemoryImpactEvidence `json:"evidence"`
	Actor             string                       `json:"actor"`
	ExpectedUpdatedAt int64                        `json:"expected_updated_at"`
}

func memoryImpactFilter(c *app.RequestContext) (store.MemoryImpactFilter, error) {
	filter := store.MemoryImpactFilter{}
	if raw := strings.TrimSpace(string(c.Query("limit"))); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 {
			return filter, errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	rawStates := strings.TrimSpace(string(c.Query("state")))
	if rawStates != "" {
		for _, raw := range strings.Split(rawStates, ",") {
			state := model.MemoryImpactState(strings.TrimSpace(raw))
			if !state.Valid() {
				return filter, errors.New("invalid memory impact state")
			}
			filter.States = append(filter.States, state)
		}
	}
	return filter, nil
}

func (h *Handler) listProjectMemoryImpacts(ctx context.Context, c *app.RequestContext) {
	filter, err := memoryImpactFilter(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	project, err := h.svc.ResolveProject(ctx, c.Param("project"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	if err := h.svc.ReconcileMemoryImpacts(ctx, project.ID); err != nil {
		writeServiceError(c, err)
		return
	}
	filter.ProjectID = project.ID
	impacts, err := h.svc.ListMemoryImpacts(ctx, filter)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, impacts)
}

func (h *Handler) listTaskMemoryImpacts(ctx context.Context, c *app.RequestContext) {
	filter, err := memoryImpactFilter(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	task, err := h.svc.GetTask(ctx, c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	filter.TaskID = task.ID
	impacts, err := h.svc.ListMemoryImpacts(ctx, filter)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, impacts)
}

func (h *Handler) listCapsuleMemoryImpacts(ctx context.Context, c *app.RequestContext) {
	filter, err := memoryImpactFilter(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	capsule, err := h.svc.GetCapsule(ctx, c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	if err := h.svc.ReconcileMemoryImpacts(ctx, capsule.ProjectID); err != nil {
		writeServiceError(c, err)
		return
	}
	filter.CapsuleID = capsule.ID
	impacts, err := h.svc.ListMemoryImpacts(ctx, filter)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, impacts)
}

func (h *Handler) updateMemoryImpact(ctx context.Context, c *app.RequestContext) {
	var req updateMemoryImpactReq
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	agent, ok := h.optionalAgent(ctx, c)
	if !ok {
		return
	}
	agentName := ""
	actor := strings.TrimSpace(req.Actor)
	if agent != nil {
		agentName, actor = agent.Name, agent.Name
	} else if actor == "" {
		actor = "web"
	}
	impact, err := h.svc.RecordMemoryImpact(ctx, service.RecordMemoryImpactInput{
		ImpactID: c.Param("id"), State: req.State, Stage: req.Stage,
		Notes: req.Notes, Evidence: req.Evidence, Actor: actor,
		AgentName: agentName, ExpectedUpdatedAt: req.ExpectedUpdatedAt,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, impact)
}
