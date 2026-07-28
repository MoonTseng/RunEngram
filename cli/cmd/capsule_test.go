package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestCapsuleWorkflowCommandsRegistered(t *testing.T) {
	if findCommand(rootCmd, "capsule") == nil {
		t.Fatal("capsule command not registered")
	}
	for _, name := range []string{"list", "create", "use", "archive", "metrics"} {
		if findCommand(capsuleCmd, name) == nil {
			t.Fatalf("capsule %s command not registered", name)
		}
	}
	if findCommand(taskCmd, "context") == nil {
		t.Fatal("task context command not registered")
	}
	for _, flag := range []string{"producer", "evidence-file"} {
		if capsuleCreateCmd.Flags().Lookup(flag) == nil {
			t.Fatalf("capsule create --%s flag not registered", flag)
		}
	}
	if capsuleUseCmd.Flags().Lookup("outcome") == nil {
		t.Fatal("capsule use --outcome flag not registered")
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
