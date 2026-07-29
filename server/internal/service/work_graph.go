package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"taskline_server/api/model"
	"taskline_server/internal/store"
)

var engineeringFlowDefinition = model.WorkflowDefinition{
	Template: model.WorkflowTemplateEngineeringFlow,
	Version:  1,
	Nodes: []model.WorkflowNodeSpec{
		{Key: "requirement-analysis", Title: "需求分析", Capability: "prd-analysis", Kind: "agent-loop"},
		{Key: "technical-design", Title: "技术方案", Capability: "technical-design", Kind: "agent-loop", DependsOn: []string{"requirement-analysis"}},
		{Key: "task-planning", Title: "任务规划", Capability: "task-planning", Kind: "agent-loop", DependsOn: []string{"technical-design"}},
		{Key: "implementation", Title: "代码实现", Capability: "coding", Kind: "agent-loop", DependsOn: []string{"task-planning"}},
		{Key: "refactor", Title: "重构优化", Capability: "refactoring", Kind: "agent-loop", DependsOn: []string{"implementation"}},
		{Key: "verification", Title: "测试验证", Capability: "verification", Kind: "evaluator", DependsOn: []string{"refactor"}},
		{Key: "code-review", Title: "独立复核", Capability: "code-review", Kind: "evaluator", DependsOn: []string{"verification"}},
		{Key: "final-gate", Title: "结果确认", Capability: "human-approval", Kind: "human", DependsOn: []string{"code-review"}},
	},
}

func buildWorkflowNodes(
	template model.WorkflowTemplate,
	custom *model.WorkflowDefinition,
) ([]*model.RunNode, int, error) {
	if template == model.WorkflowTemplateSingleLoop {
		if custom != nil && len(custom.Nodes) > 0 {
			return nil, 0, errors.New("single-loop cannot define workflow nodes")
		}
		return nil, 1, nil
	}
	definition := custom
	if definition == nil && template == model.WorkflowTemplateEngineeringFlow {
		builtin := engineeringFlowDefinition
		definition = &builtin
	}
	if definition == nil {
		return nil, 0, fmt.Errorf(
			"workflow definition required for custom template %q",
			template,
		)
	}
	if definition.Template == "" {
		definition.Template = template
	}
	if definition.Template != template {
		return nil, 0, errors.New("workflow definition template mismatch")
	}
	if err := validateWorkflowDefinition(definition); err != nil {
		return nil, 0, err
	}
	version := definition.Version
	if version == 0 {
		version = 1
	}
	nodes := make([]*model.RunNode, 0, len(definition.Nodes))
	for position, nodeSpec := range definition.Nodes {
		status := model.RunNodePending
		if len(nodeSpec.DependsOn) == 0 {
			status = model.RunNodeReady
		}
		nodes = append(nodes, &model.RunNode{
			Key: nodeSpec.Key, Title: nodeSpec.Title,
			Capability: nodeSpec.Capability, Kind: nodeSpec.Kind,
			Position: position, DependsOn: append([]string(nil), nodeSpec.DependsOn...),
			Status: status, ArtifactIDs: []string{},
		})
	}
	return nodes, version, nil
}

func validateWorkflowDefinition(definition *model.WorkflowDefinition) error {
	if definition == nil {
		return errors.New("workflow definition required")
	}
	if !definition.Template.Valid() ||
		definition.Template == model.WorkflowTemplateSingleLoop {
		return fmt.Errorf("invalid graph workflow template %q", definition.Template)
	}
	if definition.Version < 0 {
		return errors.New("workflow version cannot be negative")
	}
	if len(definition.Nodes) < 1 || len(definition.Nodes) > 32 {
		return errors.New("workflow definition requires 1 to 32 nodes")
	}
	nodeByKey := make(map[string]model.WorkflowNodeSpec, len(definition.Nodes))
	for index := range definition.Nodes {
		node := &definition.Nodes[index]
		node.Key = strings.TrimSpace(node.Key)
		node.Title = strings.TrimSpace(node.Title)
		node.Capability = strings.TrimSpace(node.Capability)
		node.Kind = strings.TrimSpace(node.Kind)
		node.DependsOn = uniqueNonEmpty(node.DependsOn)
		if !model.WorkflowTemplate(node.Key).Valid() {
			return fmt.Errorf("invalid workflow node key %q", node.Key)
		}
		if node.Title == "" || len(node.Title) > 120 {
			return fmt.Errorf("workflow node %s requires title up to 120 bytes", node.Key)
		}
		if node.Capability == "" || len(node.Capability) > 100 {
			return fmt.Errorf("workflow node %s requires capability up to 100 bytes", node.Key)
		}
		switch node.Kind {
		case "agent-loop", "evaluator", "human", "tool":
		default:
			return fmt.Errorf("workflow node %s has invalid kind %q", node.Key, node.Kind)
		}
		if _, exists := nodeByKey[node.Key]; exists {
			return fmt.Errorf("duplicate workflow node key %q", node.Key)
		}
		nodeByKey[node.Key] = *node
	}
	indegree := make(map[string]int, len(nodeByKey))
	children := make(map[string][]string, len(nodeByKey))
	for _, node := range definition.Nodes {
		for _, dependency := range node.DependsOn {
			if dependency == node.Key {
				return fmt.Errorf("workflow node %s cannot depend on itself", node.Key)
			}
			if _, exists := nodeByKey[dependency]; !exists {
				return fmt.Errorf(
					"workflow node %s depends on unknown node %s",
					node.Key,
					dependency,
				)
			}
			indegree[node.Key]++
			children[dependency] = append(children[dependency], node.Key)
		}
	}
	queue := make([]string, 0, len(nodeByKey))
	for key := range nodeByKey {
		if indegree[key] == 0 {
			queue = append(queue, key)
		}
	}
	visited := 0
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		visited++
		for _, child := range children[key] {
			indegree[child]--
			if indegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}
	if visited != len(nodeByKey) {
		return errors.New("workflow definition contains a dependency cycle")
	}
	return nil
}

type UpdateRunNodeInput struct {
	RunID            string
	AgentName        string
	NodeKey          string
	Status           model.RunNodeStatus
	Summary          string
	NextStep         string
	ArtifactIDs      []string
	Evidence         string
	InputFingerprint string
}

func (s *Service) UpdateRunNode(
	ctx context.Context,
	input UpdateRunNodeInput,
) (*model.RunNode, error) {
	input.NodeKey = strings.TrimSpace(input.NodeKey)
	input.Summary = strings.TrimSpace(input.Summary)
	input.NextStep = strings.TrimSpace(input.NextStep)
	input.Evidence = strings.TrimSpace(input.Evidence)
	input.InputFingerprint = strings.TrimSpace(input.InputFingerprint)
	if input.NodeKey == "" {
		return nil, errors.New("workflow node key required")
	}
	if !input.Status.Valid() {
		return nil, fmt.Errorf("invalid workflow node status %q", input.Status)
	}
	run, task, err := s.ownedRun(ctx, input.RunID, input.AgentName)
	if err != nil {
		return nil, err
	}
	if run.WorkflowTemplate == model.WorkflowTemplateSingleLoop {
		return nil, fmt.Errorf("%w: run has no work graph", store.ErrConflict)
	}
	if run.Status.Terminal() {
		return nil, fmt.Errorf("%w: run already finished", store.ErrConflict)
	}
	nodes, err := s.st.ListRunNodes(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	nodeByKey := make(map[string]*model.RunNode, len(nodes))
	for _, node := range nodes {
		nodeByKey[node.Key] = node
	}
	node, ok := nodeByKey[input.NodeKey]
	if !ok {
		return nil, store.ErrNotFound
	}
	if input.Status == model.RunNodeRunning ||
		input.Status == model.RunNodeWaiting ||
		input.Status == model.RunNodeCompleted ||
		input.Status == model.RunNodeSkipped {
		for _, dependency := range node.DependsOn {
			dependencyNode := nodeByKey[dependency]
			if dependencyNode == nil || !dependencyNode.Status.SatisfiesDependency() {
				return nil, fmt.Errorf(
					"%w: workflow node %s waits for %s",
					store.ErrConflict,
					node.Key,
					dependency,
				)
			}
		}
	}
	if node.Kind == "human" && input.Status == model.RunNodeSkipped {
		return nil, fmt.Errorf("%w: human gate cannot be skipped", store.ErrConflict)
	}
	if node.Kind == "human" && input.Status == model.RunNodeCompleted {
		if node.StartedAt == 0 {
			return nil, fmt.Errorf(
				"%w: human gate requires a fresh answered interrupt",
				store.ErrConflict,
			)
		}
		answered, answeredErr := s.st.HasAnsweredRunInterruptSince(
			ctx,
			run.ID,
			node.Key,
			node.StartedAt,
		)
		if answeredErr != nil {
			return nil, answeredErr
		}
		if !answered {
			return nil, fmt.Errorf(
				"%w: human gate requires a fresh answered interrupt",
				store.ErrConflict,
			)
		}
	}
	if input.Status == model.RunNodeCompleted && input.Summary == "" && node.Summary == "" {
		return nil, errors.New("completed workflow node requires summary")
	}
	if input.ArtifactIDs != nil {
		if err := s.validateTaskArtifacts(ctx, task.ID, input.ArtifactIDs); err != nil {
			return nil, err
		}
	}
	invalidateDownstream := node.Status.SatisfiesDependency() &&
		!input.Status.SatisfiesDependency()
	timestamp := nowMillis()
	if input.Status == model.RunNodeRunning && node.Status != model.RunNodeRunning {
		node.Attempt++
		if node.StartedAt == 0 {
			node.StartedAt = timestamp
		}
	}
	node.Status = input.Status
	if input.Summary != "" {
		node.Summary = input.Summary
	}
	if input.NextStep != "" || input.Status == model.RunNodeCompleted {
		node.NextStep = input.NextStep
	}
	if input.ArtifactIDs != nil {
		node.ArtifactIDs = uniqueNonEmpty(input.ArtifactIDs)
	}
	if input.Evidence != "" {
		node.Evidence = input.Evidence
	}
	if input.InputFingerprint != "" {
		node.InputFingerprint = input.InputFingerprint
	}
	if input.Status == model.RunNodeCompleted || input.Status == model.RunNodeSkipped {
		node.CompletedAt = timestamp
		if node.StartedAt == 0 {
			node.StartedAt = timestamp
		}
	} else {
		node.CompletedAt = 0
	}
	node, err = s.st.UpdateRunNode(ctx, node)
	if err != nil {
		return nil, err
	}
	if invalidateDownstream {
		if err := s.invalidateDependentNodes(ctx, run.ID, node.Key); err != nil {
			return nil, err
		}
	}
	if err := s.refreshRunNodeReadiness(ctx, run.ID); err != nil {
		return nil, err
	}
	_, err = s.appendTaskEvent(
		ctx,
		task.ID,
		string(model.RunEventNodeUpdated),
		fmt.Sprintf("Updated Work Graph stage: %s", node.Title),
		runEventDetails(run, map[string]any{
			"node_key": node.Key, "node_status": node.Status,
			"summary": node.Summary, "artifact_count": len(node.ArtifactIDs),
			"verified": node.Evidence != "",
		}),
		node.UpdatedAt,
	)
	return node, err
}

func (s *Service) validateTaskArtifacts(
	ctx context.Context,
	taskID string,
	artifactIDs []string,
) error {
	for _, artifactID := range uniqueNonEmpty(artifactIDs) {
		if doc, err := s.st.GetDoc(ctx, artifactID); err == nil {
			if doc.TaskID != taskID {
				return fmt.Errorf("%w: artifact belongs to another task", store.ErrConflict)
			}
			continue
		} else if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		if image, err := s.st.GetImage(ctx, artifactID); err == nil {
			if image.TaskID != taskID {
				return fmt.Errorf("%w: artifact belongs to another task", store.ErrConflict)
			}
			continue
		} else if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		if link, err := s.st.GetLink(ctx, artifactID); err == nil {
			if link.TaskID != taskID {
				return fmt.Errorf("%w: artifact belongs to another task", store.ErrConflict)
			}
			continue
		} else if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		return fmt.Errorf("%w: artifact %s does not exist", store.ErrNotFound, artifactID)
	}
	return nil
}

func (s *Service) invalidateDependentNodes(
	ctx context.Context,
	runID, changedNodeKey string,
) error {
	nodes, err := s.st.ListRunNodes(ctx, runID)
	if err != nil {
		return err
	}
	invalidated := map[string]bool{changedNodeKey: true}
	changed := true
	for changed {
		changed = false
		for _, node := range nodes {
			if invalidated[node.Key] {
				continue
			}
			dependsOnInvalidated := false
			for _, dependency := range node.DependsOn {
				if invalidated[dependency] {
					dependsOnInvalidated = true
					break
				}
			}
			if !dependsOnInvalidated {
				continue
			}
			invalidated[node.Key] = true
			changed = true
			node.Status = model.RunNodePending
			node.Summary = ""
			node.NextStep = ""
			node.ArtifactIDs = []string{}
			node.Evidence = ""
			node.InputFingerprint = ""
			node.StartedAt = 0
			node.CompletedAt = 0
			if _, err := s.st.UpdateRunNode(ctx, node); err != nil {
				return err
			}
		}
	}
	pending, err := s.st.ListPendingRunInterrupts(ctx, runID)
	if err != nil {
		return err
	}
	for _, interrupt := range pending {
		if !invalidated[interrupt.NodeKey] {
			continue
		}
		interrupt.Status = model.RunInterruptRejected
		interrupt.Response = "invalidated by upstream change"
		interrupt.RespondedBy = "runengram"
		interrupt.ResolvedAt = nowMillis()
		if _, err := s.st.UpdateRunInterrupt(ctx, interrupt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) refreshRunNodeReadiness(ctx context.Context, runID string) error {
	nodes, err := s.st.ListRunNodes(ctx, runID)
	if err != nil {
		return err
	}
	nodeByKey := make(map[string]*model.RunNode, len(nodes))
	for _, node := range nodes {
		nodeByKey[node.Key] = node
	}
	for _, node := range nodes {
		if node.Status != model.RunNodePending && node.Status != model.RunNodeReady {
			continue
		}
		ready := true
		for _, dependency := range node.DependsOn {
			dependencyNode := nodeByKey[dependency]
			if dependencyNode == nil || !dependencyNode.Status.SatisfiesDependency() {
				ready = false
				break
			}
		}
		nextStatus := model.RunNodePending
		if ready {
			nextStatus = model.RunNodeReady
		}
		if node.Status != nextStatus {
			node.Status = nextStatus
			if _, err := s.st.UpdateRunNode(ctx, node); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) GetRunWorkGraph(
	ctx context.Context,
	runID string,
) (*model.RunWorkGraph, error) {
	run, err := s.st.GetAgentRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	nodes, err := s.st.ListRunNodes(ctx, runID)
	if err != nil {
		return nil, err
	}
	interrupts, err := s.st.ListPendingRunInterrupts(ctx, runID)
	if err != nil {
		return nil, err
	}
	if nodes == nil {
		nodes = []*model.RunNode{}
	}
	if interrupts == nil {
		interrupts = []*model.RunInterrupt{}
	}
	graph := &model.RunWorkGraph{
		RunID: run.ID, Template: run.WorkflowTemplate, Version: run.WorkflowVersion,
		Nodes: nodes, Interrupts: interrupts, OpenInterruptCount: len(interrupts),
	}
	artifacts := make(map[string]struct{})
	resolvedNodeCount := 0
	for _, node := range nodes {
		if node.Status == model.RunNodeCompleted {
			graph.CompletedNodeCount++
		}
		if node.Status.SatisfiesDependency() {
			resolvedNodeCount++
		}
		if node.Status == model.RunNodeCompleted &&
			strings.TrimSpace(node.Evidence) != "" {
			graph.VerifiedNodeCount++
		}
		for _, artifactID := range node.ArtifactIDs {
			artifacts[artifactID] = struct{}{}
		}
	}
	graph.ArtifactCount = len(artifacts)
	if len(nodes) > 0 {
		graph.ProgressPercent = resolvedNodeCount * 100 / len(nodes)
	}
	return graph, nil
}

type CreateRunInterruptInput struct {
	RunID     string
	AgentName string
	NodeKey   string
	Kind      model.RunInterruptKind
	Prompt    string
	Options   []string
}

func (s *Service) CreateRunInterrupt(
	ctx context.Context,
	input CreateRunInterruptInput,
) (*model.RunInterrupt, error) {
	input.NodeKey = strings.TrimSpace(input.NodeKey)
	input.Prompt = strings.TrimSpace(input.Prompt)
	if !input.Kind.Valid() {
		return nil, fmt.Errorf("invalid interrupt kind %q", input.Kind)
	}
	if input.Prompt == "" {
		return nil, errors.New("interrupt prompt required")
	}
	run, task, err := s.ownedRun(ctx, input.RunID, input.AgentName)
	if err != nil {
		return nil, err
	}
	if run.Status.Terminal() {
		return nil, fmt.Errorf("%w: run already finished", store.ErrConflict)
	}
	if run.WorkflowTemplate == model.WorkflowTemplateSingleLoop {
		return nil, fmt.Errorf("%w: run has no work graph", store.ErrConflict)
	}
	node, err := s.st.GetRunNode(ctx, run.ID, input.NodeKey)
	if err != nil {
		return nil, err
	}
	if node.Status != model.RunNodeReady &&
		node.Status != model.RunNodeRunning &&
		node.Status != model.RunNodeFailed {
		return nil, fmt.Errorf(
			"%w: node %s cannot wait from status %s",
			store.ErrConflict,
			node.Key,
			node.Status,
		)
	}
	nodes, err := s.st.ListRunNodes(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	nodeByKey := make(map[string]*model.RunNode, len(nodes))
	for _, candidate := range nodes {
		nodeByKey[candidate.Key] = candidate
	}
	for _, dependency := range node.DependsOn {
		dependencyNode := nodeByKey[dependency]
		if dependencyNode == nil || !dependencyNode.Status.SatisfiesDependency() {
			return nil, fmt.Errorf(
				"%w: workflow node %s waits for %s",
				store.ErrConflict,
				node.Key,
				dependency,
			)
		}
	}
	pending, err := s.st.ListPendingRunInterrupts(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	for _, interrupt := range pending {
		if interrupt.NodeKey == input.NodeKey {
			return nil, fmt.Errorf("%w: node already waits for a response", store.ErrConflict)
		}
	}
	interrupt, err := s.st.CreateRunInterrupt(ctx, &model.RunInterrupt{
		RunID: run.ID, NodeKey: node.Key, Kind: input.Kind, Prompt: input.Prompt,
		Options: uniqueNonEmpty(input.Options), RequestedBy: input.AgentName,
	})
	if err != nil {
		return nil, err
	}
	node.Status = model.RunNodeWaiting
	node.NextStep = input.Prompt
	node.CompletedAt = 0
	if node.StartedAt == 0 {
		node.StartedAt = interrupt.CreatedAt
		node.Attempt++
	}
	if _, err := s.st.UpdateRunNode(ctx, node); err != nil {
		return nil, err
	}
	_, err = s.appendTaskEvent(
		ctx,
		task.ID,
		string(model.RunEventInterruptCreated),
		fmt.Sprintf("Waiting for input: %s", node.Title),
		runEventDetails(run, map[string]any{
			"interrupt_id": interrupt.ID, "node_key": node.Key,
			"kind": interrupt.Kind, "prompt": interrupt.Prompt,
		}),
		interrupt.CreatedAt,
	)
	return interrupt, err
}

type ResolveRunInterruptInput struct {
	InterruptID string
	Response    string
	RespondedBy string
	Reject      bool
}

func (s *Service) ResolveRunInterrupt(
	ctx context.Context,
	input ResolveRunInterruptInput,
) (*model.RunInterrupt, error) {
	input.Response = strings.TrimSpace(input.Response)
	input.RespondedBy = strings.TrimSpace(input.RespondedBy)
	if input.Response == "" {
		return nil, errors.New("interrupt response required")
	}
	if input.RespondedBy == "" {
		input.RespondedBy = actorFromContext(ctx)
	}
	interrupt, err := s.st.GetRunInterrupt(ctx, input.InterruptID)
	if err != nil {
		return nil, err
	}
	if interrupt.Status != model.RunInterruptPending {
		return nil, fmt.Errorf("%w: interrupt already resolved", store.ErrConflict)
	}
	if len(interrupt.Options) > 0 && !containsString(interrupt.Options, input.Response) {
		return nil, fmt.Errorf("response must be one of: %s", strings.Join(interrupt.Options, ", "))
	}
	run, err := s.st.GetAgentRun(ctx, interrupt.RunID)
	if err != nil {
		return nil, err
	}
	node, err := s.st.GetRunNode(ctx, run.ID, interrupt.NodeKey)
	if err != nil {
		return nil, err
	}
	if node.Kind == "human" && input.RespondedBy == interrupt.RequestedBy {
		return nil, fmt.Errorf(
			"%w: human gate must be resolved by someone other than the executing agent",
			store.ErrConflict,
		)
	}
	interrupt.Status = model.RunInterruptAnswered
	if input.Reject {
		interrupt.Status = model.RunInterruptRejected
	}
	interrupt.Response = input.Response
	interrupt.RespondedBy = input.RespondedBy
	interrupt.ResolvedAt = nowMillis()
	interrupt, err = s.st.UpdateRunInterrupt(ctx, interrupt)
	if err != nil {
		return nil, err
	}
	if node.Status == model.RunNodeWaiting {
		node.Status = model.RunNodeRunning
		node.Attempt++
		if node.StartedAt == 0 {
			node.StartedAt = interrupt.ResolvedAt
		}
		node.NextStep = ""
		if _, err := s.st.UpdateRunNode(ctx, node); err != nil {
			return nil, err
		}
	}
	_, err = s.appendTaskEvent(
		ctx,
		run.TaskID,
		string(model.RunEventInterruptResolved),
		fmt.Sprintf("Resolved input for stage: %s", node.Title),
		runEventDetails(run, map[string]any{
			"interrupt_id": interrupt.ID, "node_key": node.Key,
			"response": interrupt.Response, "status": interrupt.Status,
		}),
		interrupt.ResolvedAt,
	)
	return interrupt, err
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
