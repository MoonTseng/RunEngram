package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"cli.taskline.dev/client"
	"cli.taskline.dev/internal/output"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Track resumable coding-agent runs",
}

var runStartCmd = &cobra.Command{
	Use:   "start <task-id>",
	Short: "Start or resume an Agent run for a claimed task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentTool, _ := cmd.Flags().GetString("agent-tool")
		result, err := newClient().StartAgentRun(args[0], agentTool)
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), result, func(w io.Writer) {
			action := "Started"
			if result.Resumed {
				action = "Resumed"
			}
			fmt.Fprintf(w, "%s %s run %s for task %s\n",
				action, result.Run.AgentTool, result.Run.ID, result.Run.TaskID)
			renderAgentRun(w, result.Run)
		})
	},
}

var runEventCmd = &cobra.Command{
	Use:   "event <run-id>",
	Short: "Append a normalized Agent run event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, _ := cmd.Flags().GetString("kind")
		summary, _ := cmd.Flags().GetString("summary")
		detailsJSON, _ := cmd.Flags().GetString("details")
		details := map[string]any{}
		if detailsJSON != "" {
			if err := json.Unmarshal([]byte(detailsJSON), &details); err != nil {
				return fmt.Errorf("decode --details JSON: %w", err)
			}
		}
		for _, key := range []string{"trigger", "guidance", "scope"} {
			value, _ := cmd.Flags().GetString(key)
			if value != "" {
				details[key] = value
			}
		}
		event, err := newClient().RecordRunEvent(args[0], kind, summary, details)
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), event, func(w io.Writer) {
			fmt.Fprintf(w, "%s  %s  %s\n", event.Kind, event.Actor, event.Summary)
		})
	},
}

var runCheckpointCmd = &cobra.Command{
	Use:   "checkpoint <run-id>",
	Short: "Save compact progress for interruption-safe resume",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		status, _ := cmd.Flags().GetString("status")
		summary, _ := cmd.Flags().GetString("summary")
		nextStep, _ := cmd.Flags().GetString("next-step")
		run, err := newClient().SaveRunCheckpoint(args[0], status, summary, nextStep)
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), run, func(w io.Writer) {
			renderAgentRun(w, *run)
		})
	},
}

var runFinishCmd = &cobra.Command{
	Use:   "finish <run-id>",
	Short: "Finish an Agent run with a durable summary",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		status, _ := cmd.Flags().GetString("status")
		summary, _ := cmd.Flags().GetString("summary")
		run, err := newClient().FinishAgentRun(args[0], status, summary)
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), run, func(w io.Writer) {
			renderAgentRun(w, *run)
		})
	},
}

var runShowCmd = &cobra.Command{
	Use:   "show <run-id>",
	Short: "Show one Agent run",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		run, err := newClient().GetAgentRun(args[0])
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), run, func(w io.Writer) {
			renderAgentRun(w, *run)
		})
	},
}

var taskResumeCmd = &cobra.Command{
	Use:   "resume <task-id>",
	Short: "Read frozen context, latest checkpoint, and recent events",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		resume, err := newClient().GetTaskResumeContext(args[0])
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), resume, func(w io.Writer) {
			fmt.Fprintf(w, "Task: %s\nSnapshot: %s\n",
				resume.Snapshot.Task.Title, resume.Snapshot.ID)
			if resume.LatestRun != nil {
				fmt.Fprintln(w, "Latest run:")
				renderAgentRun(w, *resume.LatestRun)
			}
			fmt.Fprintf(w, "Recent events: %d\n", len(resume.RecentEvents))
		})
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.AddCommand(
		runStartCmd, runEventCmd, runCheckpointCmd, runFinishCmd, runShowCmd,
	)
	taskCmd.AddCommand(taskResumeCmd)

	runStartCmd.Flags().String(
		"agent-tool", "codex", "execution engine: codex|claude-code|pi|other",
	)
	runEventCmd.Flags().String(
		"kind", "", "tool.called|verification.passed|learning.discovered (required)",
	)
	runEventCmd.Flags().String("summary", "", "compact event summary (required)")
	runEventCmd.Flags().String("details", "", "JSON object with event details")
	runEventCmd.Flags().String("trigger", "", "learning trigger")
	runEventCmd.Flags().String("guidance", "", "reusable learning guidance")
	runEventCmd.Flags().String("scope", "", "learning scope")
	_ = runEventCmd.MarkFlagRequired("kind")
	_ = runEventCmd.MarkFlagRequired("summary")

	runCheckpointCmd.Flags().String("status", "running", "running|blocked")
	runCheckpointCmd.Flags().String("summary", "", "completed work and decisions (required)")
	runCheckpointCmd.Flags().String("next-step", "", "smallest concrete next action")
	_ = runCheckpointCmd.MarkFlagRequired("summary")

	runFinishCmd.Flags().String("status", "completed", "completed|failed")
	runFinishCmd.Flags().String("summary", "", "final outcome and verification (required)")
	_ = runFinishCmd.MarkFlagRequired("summary")
}

func renderAgentRun(w io.Writer, run client.AgentRun) {
	fmt.Fprintf(w, "%-36s  %-12s  %-11s  %s\n",
		run.ID, run.AgentTool, run.Status, run.Summary)
	if run.NextStep != "" {
		fmt.Fprintf(w, "Next: %s\n", run.NextStep)
	}
}
