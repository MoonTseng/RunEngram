package cmd

import "testing"

func TestRootCommandUsesRunEngramName(t *testing.T) {
	if rootCmd.Use != "runengram" {
		t.Fatalf("root command name = %q, want runengram", rootCmd.Use)
	}
}
