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
	for _, flag := range []string{"evidence-file", "memory-class"} {
		if learningPromoteCmd.Flags().Lookup(flag) == nil {
			t.Fatalf("learning promote --%s flag not registered", flag)
		}
	}
	for _, flag := range []string{"trigger", "guidance", "scope"} {
		if learningEditCmd.Flags().Lookup(flag) == nil {
			t.Fatalf("learning edit --%s flag not registered", flag)
		}
	}
}
