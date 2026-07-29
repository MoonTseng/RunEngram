package cmd

import "testing"

func TestRunLoopCommandsRegistered(t *testing.T) {
	run := findCommand(rootCmd, "run")
	if run == nil {
		t.Fatal("run command not registered")
	}
	for _, name := range []string{
		"start", "event", "checkpoint", "finish", "show",
		"graph", "node", "interrupt", "respond",
	} {
		if findCommand(run, name) == nil {
			t.Fatalf("run %s command not registered", name)
		}
	}
	resume := findCommand(taskCmd, "resume")
	if resume == nil {
		t.Fatal("task resume command not registered")
	}
	if findCommand(run, "start").Flags().Lookup("agent-tool") == nil {
		t.Fatal("run start --agent-tool flag not registered")
	}
	if findCommand(run, "start").Flags().Lookup("workflow") == nil {
		t.Fatal("run start --workflow flag not registered")
	}
}
