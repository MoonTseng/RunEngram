package main

import (
	"cli.taskline.dev/cmd"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

func main() {
	cmd.SetVersion(Version)
	cmd.SetCommit(Commit)
	cmd.Execute()
}
