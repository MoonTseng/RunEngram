package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestCapsuleWorkflowCommandsRegistered(t *testing.T) {
	if findCommand(rootCmd, "capsule") == nil {
		t.Fatal("capsule command not registered")
	}
	for _, name := range []string{"list", "create", "edit", "use", "archive", "metrics", "relate", "unrelate"} {
		if findCommand(capsuleCmd, name) == nil {
			t.Fatalf("capsule %s command not registered", name)
		}
	}
	if findCommand(taskCmd, "context") == nil {
		t.Fatal("task context command not registered")
	}
	if findCommand(taskCmd, "recall") == nil {
		t.Fatal("task recall command not registered")
	}
	for _, flag := range []string{"producer", "evidence-file", "memory-class", "trigger"} {
		if capsuleCreateCmd.Flags().Lookup(flag) == nil {
			t.Fatalf("capsule create --%s flag not registered", flag)
		}
	}
	for _, flag := range []string{
		"outcome", "stage", "notes", "evidence-kind", "evidence-ref",
		"evidence-summary", "expected-updated-at",
	} {
		if capsuleUseCmd.Flags().Lookup(flag) == nil {
			t.Fatalf("capsule use --%s flag not registered", flag)
		}
	}
	for _, flag := range []string{"type", "target-kind", "target", "note"} {
		if capsuleRelateCmd.Flags().Lookup(flag) == nil {
			t.Fatalf("capsule relate --%s flag not registered", flag)
		}
	}
	for _, flag := range []string{"title", "summary", "trigger", "scope", "evidence-file", "expected-updated-at"} {
		if capsuleEditCmd.Flags().Lookup(flag) == nil {
			t.Fatalf("capsule edit --%s flag not registered", flag)
		}
	}
}

func TestValidateCapsuleUseEvidence(t *testing.T) {
	for _, test := range []struct {
		name    string
		outcome string
		notes   string
		kind    string
		ref     string
		summary string
		wantErr bool
	}{
		{name: "applied", outcome: "used", notes: "Applied rule."},
		{name: "ignored", outcome: "ignored", notes: "Not relevant."},
		{name: "missing notes", outcome: "used", wantErr: true},
		{name: "helpful evidence", outcome: "helpful", notes: "Changed task.", kind: "task-doc", ref: "doc:test"},
		{name: "helpful missing evidence", outcome: "helpful", notes: "Changed task.", wantErr: true},
		{name: "partial evidence", outcome: "used", notes: "Changed task.", kind: "task-doc", wantErr: true},
		{name: "invalid outcome", outcome: "maybe", notes: "Unknown.", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateCapsuleUse(test.outcome, test.notes, test.kind, test.ref, test.summary)
			if test.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func findCommand(parent interface{ Commands() []*cobra.Command }, name string) *cobra.Command {
	for _, command := range parent.Commands() {
		if command.Name() == name {
			return command
		}
	}
	return nil
}
