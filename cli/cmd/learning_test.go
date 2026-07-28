package cmd

import "testing"

func TestLearningCommandsRegistered(t *testing.T) {
	if findCommand(rootCmd, "learning") == nil {
		t.Fatal("learning command not registered")
	}
	for _, name := range []string{"capture", "list", "edit", "promote", "reject"} {
		if findCommand(learningCmd, name) == nil {
			t.Fatalf("learning %s command not registered", name)
		}
	}
	for _, flag := range []string{"trigger", "guidance"} {
		if learningCaptureCmd.Flags().Lookup(flag) == nil {
			t.Fatalf("learning capture --%s flag not registered", flag)
		}
	}
	if learningPromoteCmd.Flags().Lookup("evidence-file") == nil {
		t.Fatal("learning promote --evidence-file flag not registered")
	}
	for _, flag := range []string{"trigger", "guidance", "scope"} {
		if learningEditCmd.Flags().Lookup(flag) == nil {
			t.Fatalf("learning edit --%s flag not registered", flag)
		}
	}
}
