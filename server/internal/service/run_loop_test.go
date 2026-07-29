package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"taskline_server/api/model"
	"taskline_server/internal/service"
	"taskline_server/internal/store"
)

func TestAgentRunCheckpointsResumeAndMeasureRecovery(t *testing.T) {
	ctx := service.WithActor(context.Background(), "codex")
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "run-loop", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(
		ctx,
		project.ID,
		"Migrate URL service",
		"Move callers, verify behavior, remove compatibility service",
		model.TaskTypeFeature,
		2,
		true,
		[]string{"webview"},
	)
	require.NoError(t, err)
	task, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)
	run, resumed, err := svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "codex", AgentTool: model.AgentToolCodex,
	})
	require.NoError(t, err)
	require.False(t, resumed)
	require.Equal(t, model.RunStatusRunning, run.Status)

	run, err = svc.SaveRunCheckpoint(ctx, service.SaveRunCheckpointInput{
		RunID: run.ID, AgentName: "codex", Status: model.RunStatusBlocked,
		Summary:  "Caller inventory complete; one hidden bridge still unresolved.",
		NextStep: "Trace bridge registration before deleting old service.",
	})
	require.NoError(t, err)
	require.Equal(t, model.RunStatusBlocked, run.Status)

	resumedRun, resumed, err := svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "codex", AgentTool: model.AgentToolCodex,
	})
	require.NoError(t, err)
	require.True(t, resumed)
	require.Equal(t, run.ID, resumedRun.ID)
	require.Equal(t, model.RunStatusRunning, resumedRun.Status)

	event, err := svc.RecordRunEvent(ctx, service.RecordRunEventInput{
		RunID: resumedRun.ID, AgentName: "codex",
		Kind:    model.RunEventVerificationPassed,
		Summary: "Compile and focused WebView tests passed.",
		Details: map[string]any{"command": "./gradlew test"},
	})
	require.NoError(t, err)
	require.Equal(t, model.RunEventVerificationPassed, event.Kind)

	learningEvent, err := svc.RecordRunEvent(ctx, service.RecordRunEventInput{
		RunID: resumedRun.ID, AgentName: "codex",
		Kind:    model.RunEventLearningDiscovered,
		Summary: "Captured project branch convention",
		Details: map[string]any{
			"kind":     "human-correction",
			"trigger":  "Creating a feature branch for release 7.23.0",
			"guidance": "Name branch 7.23.0_feat/<english-requirement-name>",
			"scope":    "CamScanner feature branches",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, learningEvent.Details["learning_note_id"])
	notes, err := svc.ListLearningNotes(ctx, service.LearningNoteListInput{
		TaskID: task.ID, Status: model.LearningNotePending,
	})
	require.NoError(t, err)
	require.Len(t, notes, 1)
	require.Equal(t, "Name branch 7.23.0_feat/<english-requirement-name>", notes[0].Guidance)

	completed, err := svc.FinishRun(ctx, service.FinishRunInput{
		RunID: resumedRun.ID, AgentName: "codex", Status: model.RunStatusCompleted,
		Summary: "Migration completed and verified.",
	})
	require.NoError(t, err)
	require.NotZero(t, completed.CompletedAt)

	resume, err := svc.GetTaskResumeContext(ctx, task.ID, "codex")
	require.NoError(t, err)
	require.Equal(t, completed.ID, resume.LatestRun.ID)
	require.Equal(t, "Migration completed and verified.", resume.LatestRun.Summary)
	require.NotEmpty(t, resume.RecentEvents)

	metrics, err := svc.GetLearningMetrics(ctx, project.ID)
	require.NoError(t, err)
	require.Equal(t, 1, metrics.RunCount)
	require.Equal(t, 1, metrics.CompletedRunCount)
	require.Equal(t, 1, metrics.BlockedRunCount)
	require.Equal(t, 1, metrics.ResumedRunCount)
	require.Equal(t, 1.0, metrics.RunCompletionRate)
	require.Equal(t, 1.0, metrics.RecoveryRate)
}

func TestEngineeringFlowRunCreatesDurableWorkGraphAndTypedInterrupt(t *testing.T) {
	ctx := service.WithActor(context.Background(), "codex")
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "engineering-flow-graph", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(
		ctx,
		project.ID,
		"Build a product requirement",
		"Run the existing project SOP with durable stage receipts.",
		model.TaskTypeFeature,
		2,
		true,
		[]string{"engineering-flow"},
	)
	require.NoError(t, err)
	task, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)
	specDoc := &model.Doc{
		TaskID: task.ID, Title: "Spec", StoragePath: "/tmp/engineering-flow-spec.md",
	}
	require.NoError(t, svc.AddDoc(ctx, specDoc))

	run, resumed, err := svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID:           task.ID,
		AgentName:        "codex",
		AgentTool:        model.AgentToolCodex,
		WorkflowTemplate: model.WorkflowTemplateEngineeringFlow,
	})
	require.NoError(t, err)
	require.False(t, resumed)
	require.Equal(t, model.WorkflowTemplateEngineeringFlow, run.WorkflowTemplate)
	require.Equal(t, 1, run.WorkflowVersion)

	graph, err := svc.GetRunWorkGraph(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, model.WorkflowTemplateEngineeringFlow, graph.Template)
	require.Len(t, graph.Nodes, 8)
	require.Equal(t, "requirement-analysis", graph.Nodes[0].Key)
	require.Equal(t, model.RunNodeReady, graph.Nodes[0].Status)
	require.Equal(t, model.RunNodePending, graph.Nodes[1].Status)
	require.Equal(t, 0, graph.CompletedNodeCount)
	for _, node := range graph.Nodes {
		require.NotNil(t, node.DependsOn)
		require.NotNil(t, node.ArtifactIDs)
	}

	_, err = svc.UpdateRunNode(ctx, service.UpdateRunNodeInput{
		RunID: run.ID, AgentName: "codex", NodeKey: "implementation",
		Status: model.RunNodeRunning, Summary: "Tried to skip design.",
	})
	require.Error(t, err)

	interrupt, err := svc.CreateRunInterrupt(ctx, service.CreateRunInterruptInput{
		RunID: run.ID, AgentName: "codex", NodeKey: "requirement-analysis",
		Kind:    model.RunInterruptApproval,
		Prompt:  "Confirm the structured PRD and scope.",
		Options: []string{"approve", "revise"},
	})
	require.NoError(t, err)
	require.Equal(t, model.RunInterruptPending, interrupt.Status)

	graph, err = svc.GetRunWorkGraph(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, model.RunNodeWaiting, graph.Nodes[0].Status)
	require.Equal(t, 1, graph.OpenInterruptCount)

	interrupt, err = svc.ResolveRunInterrupt(ctx, service.ResolveRunInterruptInput{
		InterruptID: interrupt.ID,
		Response:    "approve",
		RespondedBy: "yue_zeng",
	})
	require.NoError(t, err)
	require.Equal(t, model.RunInterruptAnswered, interrupt.Status)

	node, err := svc.UpdateRunNode(ctx, service.UpdateRunNodeInput{
		RunID: run.ID, AgentName: "codex", NodeKey: "requirement-analysis",
		Status:           model.RunNodeCompleted,
		Summary:          "Structured PRD confirmed.",
		NextStep:         "Design the technical solution.",
		ArtifactIDs:      []string{specDoc.ID},
		Evidence:         "User approved the structured PRD.",
		InputFingerprint: "sha256:requirement-v1",
	})
	require.NoError(t, err)
	require.Equal(t, model.RunNodeCompleted, node.Status)

	resume, err := svc.GetTaskResumeContext(ctx, task.ID, "codex")
	require.NoError(t, err)
	require.NotNil(t, resume.WorkGraph)
	require.Equal(t, 1, resume.WorkGraph.CompletedNodeCount)
	require.Equal(t, 1, resume.WorkGraph.VerifiedNodeCount)
	require.Equal(t, 1, resume.WorkGraph.ArtifactCount)
	require.Equal(t, model.RunNodeReady, resume.WorkGraph.Nodes[1].Status)
	require.Empty(t, resume.WorkGraph.Interrupts)
}

func TestCustomWorkflowDefinitionCreatesPortableGraph(t *testing.T) {
	ctx := service.WithActor(context.Background(), "codex")
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "custom-workflow", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(
		ctx, project.ID, "Publish a design note", "", model.TaskTypeDocs, 2, true, nil,
	)
	require.NoError(t, err)
	task, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)

	definition := &model.WorkflowDefinition{
		Template: "content-review",
		Version:  3,
		Nodes: []model.WorkflowNodeSpec{
			{
				Key: "draft", Title: "Draft", Capability: "writing",
				Kind: "agent-loop",
			},
			{
				Key: "fact-check", Title: "Fact check", Capability: "verification",
				Kind: "evaluator", DependsOn: []string{"draft"},
			},
			{
				Key: "publish-gate", Title: "Publish", Capability: "human-approval",
				Kind: "human", DependsOn: []string{"fact-check"},
			},
		},
	}
	run, resumed, err := svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "codex", AgentTool: model.AgentToolCodex,
		WorkflowTemplate: definition.Template, WorkflowDefinition: definition,
	})
	require.NoError(t, err)
	require.False(t, resumed)
	require.Equal(t, model.WorkflowTemplate("content-review"), run.WorkflowTemplate)
	require.Equal(t, 3, run.WorkflowVersion)

	graph, err := svc.GetRunWorkGraph(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, model.WorkflowTemplate("content-review"), graph.Template)
	require.Len(t, graph.Nodes, 3)
	require.Equal(t, model.RunNodeReady, graph.Nodes[0].Status)
	require.Equal(t, []string{"draft"}, graph.Nodes[1].DependsOn)
}

func TestCustomWorkflowDefinitionRejectsCyclesAndMissingDefinitions(t *testing.T) {
	ctx := service.WithActor(context.Background(), "codex")
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "invalid-custom-workflow", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(
		ctx, project.ID, "Invalid graph", "", model.TaskTypeFeature, 2, true, nil,
	)
	require.NoError(t, err)
	task, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)

	_, _, err = svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "codex", AgentTool: model.AgentToolCodex,
		WorkflowTemplate: "unknown-flow",
	})
	require.ErrorContains(t, err, "workflow definition required")

	_, _, err = svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "codex", AgentTool: model.AgentToolCodex,
		WorkflowTemplate: "cyclic-flow",
		WorkflowDefinition: &model.WorkflowDefinition{
			Template: "cyclic-flow",
			Nodes: []model.WorkflowNodeSpec{
				{
					Key: "first", Title: "First", Capability: "work",
					Kind: "agent-loop", DependsOn: []string{"second"},
				},
				{
					Key: "second", Title: "Second", Capability: "work",
					Kind: "agent-loop", DependsOn: []string{"first"},
				},
			},
		},
	})
	require.ErrorContains(t, err, "dependency cycle")
}

func TestEngineeringFlowFinalGateRequiresHumanDecisionAndInvalidatesDownstreamReceipts(t *testing.T) {
	ctx := service.WithActor(context.Background(), "codex")
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "engineering-flow-gates", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(
		ctx, project.ID, "Verify graph gates", "", model.TaskTypeFeature, 1, true, nil,
	)
	require.NoError(t, err)
	task, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	require.NoError(t, err)
	run, _, err := svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "codex", AgentTool: model.AgentToolCodex,
		WorkflowTemplate: model.WorkflowTemplateEngineeringFlow,
	})
	require.NoError(t, err)

	_, err = svc.UpdateRunNode(ctx, service.UpdateRunNodeInput{
		RunID: run.ID, AgentName: "codex", NodeKey: "requirement-analysis",
		Status: model.RunNodeCompleted, Summary: "Invalid artifact must fail.",
		ArtifactIDs: []string{"not-a-task-artifact"},
	})
	require.Error(t, err)

	graph, err := svc.GetRunWorkGraph(ctx, run.ID)
	require.NoError(t, err)
	for _, node := range graph.Nodes[:7] {
		status := model.RunNodeCompleted
		evidence := node.Title + " verified"
		if node.Key == "refactor" {
			status = model.RunNodeSkipped
			evidence = ""
		}
		_, err = svc.UpdateRunNode(ctx, service.UpdateRunNodeInput{
			RunID: run.ID, AgentName: "codex", NodeKey: node.Key,
			Status: status, Summary: node.Title + " complete",
			Evidence: evidence,
		})
		require.NoError(t, err)
	}

	_, err = svc.UpdateRunNode(ctx, service.UpdateRunNodeInput{
		RunID: run.ID, AgentName: "codex", NodeKey: "final-gate",
		Status: model.RunNodeCompleted, Summary: "Agent cannot approve itself.",
	})
	require.ErrorIs(t, err, store.ErrConflict)

	interrupt, err := svc.CreateRunInterrupt(ctx, service.CreateRunInterruptInput{
		RunID: run.ID, AgentName: "codex", NodeKey: "final-gate",
		Kind: model.RunInterruptApproval, Prompt: "Accept verified result?",
		Options: []string{"accept", "revise"},
	})
	require.NoError(t, err)
	_, err = svc.ResolveRunInterrupt(ctx, service.ResolveRunInterruptInput{
		InterruptID: interrupt.ID, Response: "accept", RespondedBy: "codex",
	})
	require.ErrorIs(t, err, store.ErrConflict)
	_, err = svc.ResolveRunInterrupt(ctx, service.ResolveRunInterruptInput{
		InterruptID: interrupt.ID, Response: "accept", RespondedBy: "developer",
	})
	require.NoError(t, err)
	_, err = svc.UpdateRunNode(ctx, service.UpdateRunNodeInput{
		RunID: run.ID, AgentName: "codex", NodeKey: "final-gate",
		Status: model.RunNodeCompleted, Summary: "Developer accepted.",
		Evidence: "Approval recorded by developer.",
	})
	require.NoError(t, err)
	graph, err = svc.GetRunWorkGraph(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, 7, graph.CompletedNodeCount)
	require.Equal(t, 7, graph.VerifiedNodeCount)
	require.Equal(t, 100, graph.ProgressPercent)
	require.NotNil(t, graph.Interrupts)

	_, err = svc.UpdateRunNode(ctx, service.UpdateRunNodeInput{
		RunID: run.ID, AgentName: "codex", NodeKey: "requirement-analysis",
		Status: model.RunNodeRunning, Summary: "Requirement changed.",
	})
	require.NoError(t, err)
	graph, err = svc.GetRunWorkGraph(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, model.RunNodeRunning, graph.Nodes[0].Status)
	for _, node := range graph.Nodes[1:] {
		require.Equal(t, model.RunNodePending, node.Status)
		require.Empty(t, node.Evidence)
		require.Empty(t, node.ArtifactIDs)
	}
	require.Equal(t, 0, graph.CompletedNodeCount)
	require.Equal(t, 0, graph.VerifiedNodeCount)
	require.Equal(t, 0, graph.ArtifactCount)

	for _, node := range graph.Nodes[:7] {
		status := model.RunNodeCompleted
		if node.Key == "refactor" {
			status = model.RunNodeSkipped
		}
		_, err = svc.UpdateRunNode(ctx, service.UpdateRunNodeInput{
			RunID: run.ID, AgentName: "codex", NodeKey: node.Key,
			Status: status, Summary: node.Title + " rerun complete",
		})
		require.NoError(t, err)
	}
	_, err = svc.UpdateRunNode(ctx, service.UpdateRunNodeInput{
		RunID: run.ID, AgentName: "codex", NodeKey: "final-gate",
		Status: model.RunNodeCompleted, Summary: "Old approval must not apply.",
	})
	require.ErrorIs(t, err, store.ErrConflict)
}

func TestAgentRunRequiresLiveTaskOwnership(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, err := svc.CreateProject(ctx, "run-owner", "")
	require.NoError(t, err)
	task, err := svc.CreateTask(
		ctx, project.ID, "Owned work", "", model.TaskTypeFeature, 0, true, nil,
	)
	require.NoError(t, err)

	_, _, err = svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "codex", AgentTool: model.AgentToolCodex,
	})
	require.ErrorIs(t, err, store.ErrConflict)

	_, err = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "alice"})
	require.NoError(t, err)
	_, _, err = svc.StartOrResumeRun(ctx, service.StartRunInput{
		TaskID: task.ID, AgentName: "bob", AgentTool: model.AgentToolClaudeCode,
	})
	require.ErrorIs(t, err, store.ErrConflict)
}
