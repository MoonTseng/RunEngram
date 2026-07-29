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
	rootCmd.AddCommand(learningCmd)
	learningCmd.AddCommand(
		learningCaptureCmd,
		learningListCmd,
		learningEditCmd,
		learningPromoteCmd,
		learningRejectCmd,
	)

	learningCaptureCmd.Flags().StringP("project", "p", "", "project id or name (or $TASKLINE_PROJECT)")
	learningCaptureCmd.Flags().String("kind", "human-correction", "human-correction|agent-recovery")
	learningCaptureCmd.Flags().String("trigger", "", "what failed or required correction (required)")
	learningCaptureCmd.Flags().String("guidance", "", "reusable guidance that fixed the run (required)")
	learningCaptureCmd.Flags().String("scope", "", "where this guidance applies")
	learningCaptureCmd.Flags().StringArray("label", nil, "label (repeatable)")
	learningCaptureCmd.Flags().StringArray("fingerprint", nil, "code/module/tool fingerprint (repeatable)")
	learningCaptureCmd.Flags().String("producer", "codex", "producer: codex|claude-code|other")
	_ = learningCaptureCmd.MarkFlagRequired("trigger")
	_ = learningCaptureCmd.MarkFlagRequired("guidance")

	learningListCmd.Flags().StringP("project", "p", "", "project id or name (or $TASKLINE_PROJECT)")
	learningListCmd.Flags().String("task", "", "source task id")
	learningListCmd.Flags().String("status", "pending", "pending|promoted|rejected|all")
	learningListCmd.Flags().Int("limit", 0, "maximum notes to return")

	learningEditCmd.Flags().String("trigger", "", "corrected trigger condition (required)")
	learningEditCmd.Flags().String("guidance", "", "corrected reusable guidance (required)")
	learningEditCmd.Flags().String("scope", "", "corrected applicability scope")
	_ = learningEditCmd.MarkFlagRequired("trigger")
	_ = learningEditCmd.MarkFlagRequired("guidance")

	learningPromoteCmd.Flags().String("evidence-file", "", "markdown verification evidence (required)")
	learningPromoteCmd.Flags().String(
		"memory-class",
		"experience",
		"memory class: experience|project-rule",
	)
	_ = learningPromoteCmd.MarkFlagRequired("evidence-file")

	learningRejectCmd.Flags().String("reason", "", "why candidate should not be reused (required)")
	_ = learningRejectCmd.MarkFlagRequired("reason")
}

var learningCmd = &cobra.Command{
	Use:   "learning",
	Short: "Capture corrections and promote verified engineering memory",
}

var learningCaptureCmd = &cobra.Command{
	Use:   "capture <task-id>",
	Short: "Capture a correction or recovery from an active agent run",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireIdentity(); err != nil {
			return err
		}
		project, err := learningProject(cmd)
		if err != nil {
			return err
		}
		kind, _ := cmd.Flags().GetString("kind")
		trigger, _ := cmd.Flags().GetString("trigger")
		guidance, _ := cmd.Flags().GetString("guidance")
		scope, _ := cmd.Flags().GetString("scope")
		labels, _ := cmd.Flags().GetStringArray("label")
		fingerprints, _ := cmd.Flags().GetStringArray("fingerprint")
		producer, _ := cmd.Flags().GetString("producer")
		note, err := newClient().CaptureLearningNote(project, client.CaptureLearningNoteInput{
			SourceTaskID: args[0], Kind: kind, Trigger: trigger, Guidance: guidance,
			Scope: scope, Labels: labels, Fingerprints: fingerprints, Producer: producer,
		})
		if err != nil {
			return err
		}
		return renderLearningNote(note)
	},
}

var learningListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending or resolved learning candidates",
	RunE: func(cmd *cobra.Command, _ []string) error {
		taskID, _ := cmd.Flags().GetString("task")
		project := resolveProjectFlag(cmd)
		if taskID == "" && project == "" {
			return errors.New("project or task required (--project, $TASKLINE_PROJECT, or --task)")
		}
		status, _ := cmd.Flags().GetString("status")
		if status == "all" {
			status = ""
		}
		limit, _ := cmd.Flags().GetInt("limit")
		notes, err := newClient().ListLearningNotes(project, taskID, status, limit)
		if err != nil {
			return err
		}
		return output.Render(os.Stdout, output.Resolve(formatFlag), map[string]any{
			"learning_notes": notes,
		}, func(w io.Writer) {
			renderLearningNoteTable(w, notes)
		})
	},
}

var learningPromoteCmd = &cobra.Command{
	Use:   "promote <note-id>",
	Short: "Promote verified candidate into recalled project memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireIdentity(); err != nil {
			return err
		}
		evidencePath, _ := cmd.Flags().GetString("evidence-file")
		evidence, err := os.ReadFile(evidencePath)
		if err != nil {
			return fmt.Errorf("read evidence file: %w", err)
		}
		memoryClass, _ := cmd.Flags().GetString("memory-class")
		note, err := newClient().PromoteLearningNote(args[0], string(evidence), memoryClass)
		if err != nil {
			return err
		}
		return renderLearningNote(note)
	},
}

var learningEditCmd = &cobra.Command{
	Use:   "edit <note-id>",
	Short: "Correct a pending learning candidate before promotion",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireIdentity(); err != nil {
			return err
		}
		trigger, _ := cmd.Flags().GetString("trigger")
		guidance, _ := cmd.Flags().GetString("guidance")
		scope, _ := cmd.Flags().GetString("scope")
		note, err := newClient().UpdateLearningNote(args[0], client.UpdateLearningNoteInput{
			Trigger: trigger, Guidance: guidance, Scope: scope,
		})
		if err != nil {
			return err
		}
		return renderLearningNote(note)
	},
}

var learningRejectCmd = &cobra.Command{
	Use:   "reject <note-id>",
	Short: "Reject candidate that should not enter recalled memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := requireIdentity(); err != nil {
			return err
		}
		reason, _ := cmd.Flags().GetString("reason")
		note, err := newClient().RejectLearningNote(args[0], reason)
		if err != nil {
			return err
		}
		return renderLearningNote(note)
	},
}

func renderLearningNote(note *client.LearningNote) error {
	return output.Render(os.Stdout, output.Resolve(formatFlag), note, func(w io.Writer) {
		renderLearningNoteTable(w, []client.LearningNote{*note})
	})
}

func renderLearningNoteTable(w io.Writer, notes []client.LearningNote) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTATUS\tKIND\tPRODUCER\tTRIGGER\tGUIDANCE")
	for _, note := range notes {
		fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%s\t%s\n",
			shortID(note.ID),
			note.Status,
			note.Kind,
			note.Producer,
			trimRune(strings.ReplaceAll(note.Trigger, "\n", " "), 42),
			trimRune(strings.ReplaceAll(note.Guidance, "\n", " "), 56),
		)
	}
	_ = tw.Flush()
}

func learningProject(cmd *cobra.Command) (string, error) {
	project := resolveProjectFlag(cmd)
	if project == "" {
		return "", errors.New("project required (--project or $TASKLINE_PROJECT)")
	}
	return project, nil
}

func resolveProjectFlag(cmd *cobra.Command) string {
	flagValue, _ := cmd.Flags().GetString("project")
	return resolveProject(flagValue)
}
