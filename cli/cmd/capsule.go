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
	capsuleCmd.AddCommand(capsuleListCmd, capsuleCreateCmd, capsuleUseCmd, capsuleArchiveCmd, capsuleMetricsCmd)
	taskCmd.AddCommand(taskContextCmd)

	for _, command := range []*cobra.Command{capsuleListCmd, capsuleCreateCmd, capsuleMetricsCmd} {
		command.Flags().StringP("project", "p", "", "project id or name (or $TASKLINE_PROJECT)")
	}
	capsuleListCmd.Flags().String("query", "", "search title, summary, scope, labels, and fingerprints")
	capsuleListCmd.Flags().String("status", "active", "capsule status: active|stale|archived|all")

	capsuleCreateCmd.Flags().String("source-task", "", "task that produced this knowledge")
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

	capsuleUseCmd.Flags().String("task", "", "task using this capsule (required)")
	capsuleUseCmd.Flags().String("outcome", "helpful", "used|helpful|rejected|stale")
	capsuleUseCmd.Flags().String("notes", "", "short outcome note")
	_ = capsuleUseCmd.MarkFlagRequired("task")
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
			fmt.Fprintf(w, "Task: %s\nSnapshot: %s\nSuggested capsules: %d\n",
				snapshot.Task.Title, snapshot.ID, len(snapshot.SuggestedCapsules))
			renderCapsuleTable(w, snapshot.SuggestedCapsules)
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
		sourceTask, _ := cmd.Flags().GetString("source-task")
		labels, _ := cmd.Flags().GetStringArray("label")
		fingerprints, _ := cmd.Flags().GetStringArray("fingerprint")
		producer, _ := cmd.Flags().GetString("producer")
		capsule, err := newClient().CreateCapsule(project, client.CreateCapsuleInput{
			SourceTaskID: sourceTask, Title: title, Summary: summary, Scope: scope,
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

var capsuleUseCmd = &cobra.Command{
	Use:   "use <capsule-id>",
	Short: "Record whether recalled knowledge helped a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID, _ := cmd.Flags().GetString("task")
		outcome, _ := cmd.Flags().GetString("outcome")
		notes, _ := cmd.Flags().GetString("notes")
		usage, err := newClient().RecordCapsuleUsage(args[0], taskID, outcome, notes)
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), usage, func(w io.Writer) {
			fmt.Fprintf(w, "Capsule %s → task %s: %s\n", usage.CapsuleID, usage.TaskID, usage.Outcome)
		})
	},
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
			fmt.Fprintf(w, "Tasks with context: %d · tasks reusing memory: %d\n", metrics.SnapshotTaskCount, metrics.ReusedTaskCount)
			fmt.Fprintf(w, "Helpful: %d · rejected: %d · helpful rate: %.0f%%\n",
				metrics.HelpfulCount, metrics.RejectedCount, metrics.HelpfulRate*100)
		})
	},
}

func renderCapsuleTable(w io.Writer, capsules []client.ExplorationCapsule) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTATUS\tPRODUCER\tUSES\tTITLE\tSUMMARY")
	for _, capsule := range capsules {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\n",
			shortID(capsule.ID), capsule.Status, capsule.Producer, capsule.UseCount,
			trimRune(capsule.Title, 32), trimRune(strings.ReplaceAll(capsule.Summary, "\n", " "), 56))
	}
	_ = tw.Flush()
}

func capsuleProject(cmd *cobra.Command) (string, error) {
	flagValue, _ := cmd.Flags().GetString("project")
	project := resolveProject(flagValue)
	if project == "" {
		return "", errors.New("project required (--project or $TASKLINE_PROJECT)")
	}
	return project, nil
}
