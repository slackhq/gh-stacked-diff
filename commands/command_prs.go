package commands

import (
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createPrsCommand(appConfig util.AppConfig) *cobra.Command {
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
			util.ExecuteOrDie(util.ExecuteOptions{Io: appConfig.Io, Retries: util.GhRetries},
				"gh", "pr", "list", "--author", "@me")
		},
	}
}
