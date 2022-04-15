package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func versionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the SpinUp version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(fmt.Sprintf("spinup version: %s", version))
		},
	}
}