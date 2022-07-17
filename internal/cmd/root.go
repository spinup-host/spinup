package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/spinup-host/spinup/build"
)

var rootCmd = &cobra.Command{
	Use:              "spinup",
	Short:            "Spinup CLI",
	TraverseChildren: true,
}

func Execute(ctx context.Context, buildInfo build.Info) error {
	rootCmd.AddCommand(versionCmd(buildInfo))
	rootCmd.AddCommand(startCmd())

	return rootCmd.ExecuteContext(ctx)
}
