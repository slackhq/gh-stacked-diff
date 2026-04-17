package commands

import (
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Outputs version number",
		Long:  "Outputs the version number.",
		Args:  cobra.NoArgs,
		Annotations: map[string]string{
			"defaultLogLevel": "error",
		},
		Run: func(cmd *cobra.Command, args []string) {
			util.Fprintln(util.GetAppConfig().Io.Out, "Version "+util.CurrentVersion+util.VersionSuffix())
		},
	}
}
