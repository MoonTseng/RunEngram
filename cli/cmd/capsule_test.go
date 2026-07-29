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
	if capsuleUseCmd.Flags().Lookup("outcome") == nil {
		t.Fatal("capsule use --outcome flag not registered")
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

func findCommand(parent interface{ Commands() []*cobra.Command }, name string) *cobra.Command {
	for _, command := range parent.Commands() {
		if command.Name() == name {
			return command
		}
	}
	return nil
}
