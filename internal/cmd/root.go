package cmd

import (
	"context"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:              "spinup",
	Short:            "Spinup CLI",
	TraverseChildren: true,
}

func Execute(ctx context.Context, apiVersion string) error {
	rootCmd.AddCommand(versionCmd(apiVersion))
	rootCmd.AddCommand(startCmd())

	return rootCmd.ExecuteContext(ctx)
}
