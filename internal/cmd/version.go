package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/spinup-host/spinup/build"
)

func versionCmd(bi build.Info) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the SpinUp version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(fmt.Sprintf(
				"Spinup version: %s.\nBuilt from: %s.\nCommit hash: %s",
				build.Version,
				build.Branch,
				build.FullCommit),
			)
		},
	}
}
