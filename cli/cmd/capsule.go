package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"cli.taskline.dev/client"
	"cli.taskline.dev/internal/output"
)

func init() {
	rootCmd.AddCommand(capsuleCmd)
	capsuleCmd.AddCommand(
		capsuleListCmd,
		capsuleCreateCmd,
		capsuleEditCmd,
		capsuleUseCmd,
		capsuleArchiveCmd,
		capsuleMetricsCmd,
		capsuleRelateCmd,
		capsuleUnrelateCmd,
	)
	taskCmd.AddCommand(taskContextCmd, taskRecallCmd)

	for _, command := range []*cobra.Command{capsuleListCmd, capsuleCreateCmd, capsuleMetricsCmd} {
		command.Flags().StringP("project", "p", "", "project id or name (or $TASKLINE_PROJECT)")
	}
	capsuleListCmd.Flags().String("query", "", "search title, summary, scope, labels, and fingerprints")
	capsuleListCmd.Flags().String("status", "active", "capsule status: active|stale|archived|all")

	capsuleCreateCmd.Flags().String("source-task", "", "task that produced this knowledge")
	capsuleCreateCmd.Flags().String("memory-class", "experience", "experience|project-rule")
	capsuleCreateCmd.Flags().String("trigger", "", "when this knowledge should apply")
	capsuleCreateCmd.Flags().String("title", "", "capsule title (required)")
	capsuleCreateCmd.Flags().String("summary", "", "reusable finding (required)")
	capsuleCreateCmd.Flags().String("scope", "", "where this finding applies")
	capsuleCreateCmd.Flags().String("evidence-file", "", "markdown evidence file (required)")
	capsuleCreateCmd.Flags().StringArray("label", nil, "label (repeatable)")
	capsuleCreateCmd.Flags().StringArray("fingerprint", nil, "code/module fingerprint (repeatable)")
	capsuleCreateCmd.Flags().String("producer", "codex", "producer: codex|claude-code|other")
	_ = capsuleCreateCmd.MarkFlagRequired("title")
	_ = capsuleCreateCmd.MarkFlagRequired("summary")
	_ = capsuleCreateCmd.MarkFlagRequired("evidence-file")

	capsuleEditCmd.Flags().String("title", "", "corrected memory title")
	capsuleEditCmd.Flags().String("summary", "", "corrected reusable finding")
	capsuleEditCmd.Flags().String("trigger", "", "corrected recall trigger")
	capsuleEditCmd.Flags().String("scope", "", "corrected applicability")
	capsuleEditCmd.Flags().String("evidence-file", "", "replacement markdown evidence file")
	capsuleEditCmd.Flags().Int64("expected-updated-at", 0, "last observed updated_at value (required)")
	_ = capsuleEditCmd.MarkFlagRequired("expected-updated-at")

	taskRecallCmd.Flags().String("query", "", "current action, error, or phase (required)")
	_ = taskRecallCmd.MarkFlagRequired("query")

	capsuleUseCmd.Flags().String("task", "", "task using this capsule (required)")
	capsuleUseCmd.Flags().String("outcome", "helpful", "used|ignored|helpful|rejected|stale")
	capsuleUseCmd.Flags().String("stage", "", "task stage where memory affected work")
	capsuleUseCmd.Flags().String("notes", "", "what changed because of this memory")
	capsuleUseCmd.Flags().String("evidence-kind", "", "command|task-doc|task-event|link|code-reference|observation")
	capsuleUseCmd.Flags().String("evidence-ref", "", "stable document, event, command, URL, or code reference")
	capsuleUseCmd.Flags().String("evidence-summary", "", "observed result")
	capsuleUseCmd.Flags().Int64("expected-updated-at", 0, "last observed impact updated_at when correcting an outcome")
	_ = capsuleUseCmd.MarkFlagRequired("task")

	capsuleRelateCmd.Flags().String("type", "", "derived-from|validated-by|applies-to|supersedes|conflicts-with|caused-by")
	capsuleRelateCmd.Flags().String("target-kind", "capsule", "capsule|task|artifact|scope")
	capsuleRelateCmd.Flags().String("target", "", "target capsule, task, artifact, or scope")
	capsuleRelateCmd.Flags().String("note", "", "why this relation exists")
	_ = capsuleRelateCmd.MarkFlagRequired("type")
	_ = capsuleRelateCmd.MarkFlagRequired("target")
}

var capsuleCmd = &cobra.Command{
	Use:   "capsule",
	Short: "Manage verified reusable engineering memory",
}

var taskContextCmd = &cobra.Command{
	Use:   "context <task-id>",
	Short: "Freeze and read task-start context with recalled capsules",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		snapshot, err := newClient().GetTaskContext(args[0])
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), snapshot, func(w io.Writer) {
			fmt.Fprintf(
				w,
				"Task: %s\nSnapshot: %s\nContext revision: %s\nProject rules: %d\nRelevant experiences: %d\n",
				snapshot.Task.Title,
				snapshot.ID,
				snapshot.ContextRevision,
				len(snapshot.ProjectRules),
				len(snapshot.SuggestedCapsules),
			)
			renderCapsuleTable(w, snapshot.ProjectRules)
			renderCapsuleTable(w, snapshot.SuggestedCapsules)
			renderMemoryExplanations(w, snapshot.Explanations)
		})
	},
}

var taskRecallCmd = &cobra.Command{
	Use:     "recall <task-id>",
	Aliases: []string{"prime"},
	Short:   "Recall verified memory for the current action, error, or phase",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireIdentity(); err != nil {
			return err
		}
		query, _ := cmd.Flags().GetString("query")
		recall, err := newClient().RecallTaskMemory(args[0], query)
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), recall, func(w io.Writer) {
			fmt.Fprintf(
				w,
				"Query: %s\nContext revision: %s\nProject rules: %d\nRelevant experiences: %d\n",
				recall.Query,
				recall.ContextRevision,
				len(recall.ProjectRules),
				len(recall.SuggestedCapsules),
			)
			renderCapsuleTable(w, recall.ProjectRules)
			renderCapsuleTable(w, recall.SuggestedCapsules)
			renderMemoryExplanations(w, recall.Explanations)
		})
	},
}

var capsuleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List reusable engineering memory",
	RunE: func(cmd *cobra.Command, _ []string) error {
		project, err := capsuleProject(cmd)
		if err != nil {
			return err
		}
		query, _ := cmd.Flags().GetString("query")
		status, _ := cmd.Flags().GetString("status")
		if status == "all" {
			status = ""
		}
		capsules, err := newClient().ListCapsules(project, query, status)
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), map[string]any{"capsules": capsules}, func(w io.Writer) {
			renderCapsuleTable(w, capsules)
		})
	},
}

var capsuleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Promote verified exploration into reusable memory",
	RunE: func(cmd *cobra.Command, _ []string) error {
		project, err := capsuleProject(cmd)
		if err != nil {
			return err
		}
		evidencePath, _ := cmd.Flags().GetString("evidence-file")
		evidence, err := os.ReadFile(evidencePath)
		if err != nil {
			return fmt.Errorf("read evidence file: %w", err)
		}
		title, _ := cmd.Flags().GetString("title")
		summary, _ := cmd.Flags().GetString("summary")
		scope, _ := cmd.Flags().GetString("scope")
		memoryClass, _ := cmd.Flags().GetString("memory-class")
		trigger, _ := cmd.Flags().GetString("trigger")
		sourceTask, _ := cmd.Flags().GetString("source-task")
		labels, _ := cmd.Flags().GetStringArray("label")
		fingerprints, _ := cmd.Flags().GetStringArray("fingerprint")
		producer, _ := cmd.Flags().GetString("producer")
		capsule, err := newClient().CreateCapsule(project, client.CreateCapsuleInput{
			SourceTaskID: sourceTask, MemoryClass: memoryClass, Trigger: trigger,
			Title: title, Summary: summary, Scope: scope,
			Evidence: string(evidence), Labels: labels, Fingerprints: fingerprints,
			Producer: producer,
		})
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), capsule, func(w io.Writer) {
			renderCapsuleTable(w, []client.ExplorationCapsule{*capsule})
		})
	},
}

var capsuleEditCmd = &cobra.Command{
	Use:   "edit <capsule-id>",
	Short: "Correct promoted memory without overwriting a concurrent review",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		expectedUpdatedAt, _ := cmd.Flags().GetInt64("expected-updated-at")
		input := client.UpdateCapsuleInput{ExpectedUpdatedAt: expectedUpdatedAt}
		changed := false
		for name, target := range map[string]**string{
			"title": &input.Title, "summary": &input.Summary,
			"trigger": &input.Trigger, "scope": &input.Scope,
		} {
			if !cmd.Flags().Changed(name) {
				continue
			}
			value, _ := cmd.Flags().GetString(name)
			*target = &value
			changed = true
		}
		if cmd.Flags().Changed("evidence-file") {
			evidencePath, _ := cmd.Flags().GetString("evidence-file")
			evidence, err := os.ReadFile(evidencePath)
			if err != nil {
				return fmt.Errorf("read evidence file: %w", err)
			}
			value := string(evidence)
			input.Evidence = &value
			changed = true
		}
		if !changed {
			return errors.New("set at least one corrected memory field")
		}
		capsule, err := newClient().UpdateCapsule(args[0], input)
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), capsule, func(w io.Writer) {
			renderCapsuleTable(w, []client.ExplorationCapsule{*capsule})
		})
	},
}

var capsuleUseCmd = &cobra.Command{
	Use:   "use <capsule-id>",
	Short: "Record whether recalled knowledge helped a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID, _ := cmd.Flags().GetString("task")
		outcome, _ := cmd.Flags().GetString("outcome")
		stage, _ := cmd.Flags().GetString("stage")
		notes, _ := cmd.Flags().GetString("notes")
		evidenceKind, _ := cmd.Flags().GetString("evidence-kind")
		evidenceRef, _ := cmd.Flags().GetString("evidence-ref")
		evidenceSummary, _ := cmd.Flags().GetString("evidence-summary")
		expectedUpdatedAt, _ := cmd.Flags().GetInt64("expected-updated-at")
		if err := validateCapsuleUse(outcome, notes, evidenceKind, evidenceRef, evidenceSummary); err != nil {
			return err
		}
		input := client.RecordCapsuleUsageInput{
			TaskID: taskID, Outcome: outcome, Stage: stage, Notes: notes,
			ExpectedUpdatedAt: expectedUpdatedAt,
		}
		if strings.TrimSpace(evidenceKind) != "" {
			input.Evidence = []client.MemoryImpactEvidence{{
				Kind: evidenceKind, Ref: evidenceRef, Summary: evidenceSummary,
			}}
		}
		impact, err := newClient().RecordCapsuleUsage(args[0], input)
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), impact, func(w io.Writer) {
			fmt.Fprintf(w, "Capsule %s → task %s: %s\n", impact.CapsuleID, impact.TaskID, impact.State)
		})
	},
}

func validateCapsuleUse(outcome, notes, evidenceKind, evidenceRef, evidenceSummary string) error {
	outcome = strings.TrimSpace(outcome)
	switch outcome {
	case "used", "ignored", "helpful", "rejected", "stale":
	default:
		return fmt.Errorf("invalid outcome %q; use used|ignored|helpful|rejected|stale", outcome)
	}
	if strings.TrimSpace(notes) == "" {
		return errors.New("--notes is required")
	}
	hasKind := strings.TrimSpace(evidenceKind) != ""
	hasResult := strings.TrimSpace(evidenceRef) != "" || strings.TrimSpace(evidenceSummary) != ""
	if hasKind != hasResult {
		return errors.New("evidence requires --evidence-kind and --evidence-ref or --evidence-summary")
	}
	if outcome == "helpful" || outcome == "rejected" || outcome == "stale" {
		if !hasKind || !hasResult {
			return errors.New("final outcome requires evidence")
		}
	}
	return nil
}

var capsuleArchiveCmd = &cobra.Command{
	Use:   "archive <capsule-id>",
	Short: "Archive knowledge that should no longer be recalled",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		capsule, err := newClient().UpdateCapsuleStatus(args[0], "archived")
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), capsule, func(w io.Writer) {
			renderCapsuleTable(w, []client.ExplorationCapsule{*capsule})
		})
	},
}

var capsuleRelateCmd = &cobra.Command{
	Use:   "relate <capsule-id>",
	Short: "Connect memory to evidence, scope, or other memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		relationType, _ := cmd.Flags().GetString("type")
		targetKind, _ := cmd.Flags().GetString("target-kind")
		target, _ := cmd.Flags().GetString("target")
		note, _ := cmd.Flags().GetString("note")
		relation, err := newClient().CreateMemoryRelation(args[0], client.CreateMemoryRelationInput{
			Type: relationType, TargetKind: targetKind, TargetRef: target, Note: note,
		})
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), relation, func(w io.Writer) {
			fmt.Fprintf(
				w,
				"%s --%s--> %s:%s\n",
				shortID(relation.SourceCapsuleID),
				relation.Type,
				relation.TargetKind,
				relation.TargetRef,
			)
		})
	},
}

var capsuleUnrelateCmd = &cobra.Command{
	Use:   "unrelate <relation-id>",
	Short: "Remove a memory relation without deleting either memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if err := newClient().DeleteMemoryRelation(args[0]); err != nil {
			return err
		}
		return output.Render(
			os.Stdout,
			output.Resolve(formatFlag),
			map[string]any{"deleted": true, "relation_id": args[0]},
			func(w io.Writer) {
				fmt.Fprintf(w, "Deleted relation %s\n", args[0])
			},
		)
	},
}

var capsuleMetricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Show observed engineering-memory reuse",
	RunE: func(cmd *cobra.Command, _ []string) error {
		project, err := capsuleProject(cmd)
		if err != nil {
			return err
		}
		metrics, err := newClient().GetLearningMetrics(project)
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), metrics, func(w io.Writer) {
			fmt.Fprintf(w, "Capsules: %d active / %d total\n", metrics.ActiveCapsuleCount, metrics.CapsuleCount)
			fmt.Fprintf(w, "Learning notes: %d pending / %d promoted / %d rejected\n",
				metrics.PendingNoteCount, metrics.PromotedNoteCount, metrics.RejectedNoteCount)
			fmt.Fprintf(w, "Promotion rate: %.0f%%\n", metrics.PromotionRate*100)
			fmt.Fprintf(w, "Tasks with context: %d · tasks reusing memory: %d\n", metrics.SnapshotTaskCount, metrics.ReusedTaskCount)
			fmt.Fprintf(w, "Helpful: %d · rejected: %d · helpful rate: %.0f%%\n",
				metrics.HelpfulCount, metrics.RejectedCount, metrics.HelpfulRate*100)
		})
	},
}

func renderCapsuleTable(w io.Writer, capsules []client.ExplorationCapsule) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tCLASS\tVALIDATION\tCONFIDENCE\tUSES\tTITLE\tSUMMARY")
	for _, capsule := range capsules {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%.0f%%\t%d\t%s\t%s\n",
			shortID(capsule.ID), capsule.MemoryClass, capsule.Validation, capsule.Confidence*100, capsule.UseCount,
			trimRune(capsule.Title, 32), trimRune(strings.ReplaceAll(capsule.Summary, "\n", " "), 56))
	}
	_ = tw.Flush()
}

func renderMemoryExplanations(w io.Writer, explanations []client.MemoryRecallExplanation) {
	if len(explanations) == 0 {
		return
	}
	fmt.Fprintln(w, "Why recalled:")
	for _, explanation := range explanations {
		reasons := make([]string, 0, len(explanation.Reasons))
		for _, reason := range explanation.Reasons {
			value := reason.Code
			if reason.Value != "" {
				value += "=" + reason.Value
			}
			reasons = append(reasons, value)
		}
		fmt.Fprintf(
			w,
			"- %s score=%.2f reasons=[%s]",
			shortID(explanation.CapsuleID),
			explanation.Score,
			strings.Join(reasons, ", "),
		)
		if len(explanation.Warnings) > 0 {
			fmt.Fprintf(w, " warnings=[%s]", strings.Join(explanation.Warnings, ", "))
		}
		fmt.Fprintln(w)
	}
}

func capsuleProject(cmd *cobra.Command) (string, error) {
	flagValue, _ := cmd.Flags().GetString("project")
	project := resolveProject(flagValue)
	if project == "" {
		return "", errors.New("project required (--project or $TASKLINE_PROJECT)")
	}
	return project, nil
}
