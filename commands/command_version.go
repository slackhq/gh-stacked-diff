package commands

import (
	"strings"

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
			"skipRepoCheck":   "true",
		},
		Run: func(cmd *cobra.Command, args []string) {
			var stableSuffix string
			if strings.TrimSpace(util.CurrentVersion) == strings.TrimSpace(util.StableVersion) {
				stableSuffix = " (stable)"
			} else {
				stableSuffix = " (preview)"
			}
			util.Fprintln(util.GetAppConfig().Io.Out, "Version "+util.CurrentVersion+stableSuffix)
		},
	}
}
