package commands

import (
	"fmt"

	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createPrsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "prs",
		Short: "Lists all Pull Requests you have open.",
		Long: "Lists all Pull Requests you have open.\n" +
			"\n" +
			"You must be logged-in, via \"gh auth login\"",
		Args: cobra.NoArgs,
		Annotations: map[string]string{
			"defaultLogLevel": "error",
		},
		Run: func(cmd *cobra.Command, args []string) {
			out := util.ExecuteOrDieTrimmed(util.ExecuteOptions{Retries: util.GhRetries, EnvironmentVariables: []string{"GH_PAGER=", "GH_FORCE_TTY=true"}},
				"gh", "pr", "list", "--author", "@me")
			fmt.Fprintln(util.GetAppConfig().Io.Out, out)
		},
	}
}
