package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() { rootCmd.AddCommand(versionCmd) }

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CLI version + commit",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("taskline %s (%s)\n", version, commit)
	},
}
