package commands

import (
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createBranchNameCommand(appConfig util.AppConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch-name [commitIndicator]",
		Short: "Outputs branch name of commit",
		Long: "Outputs the branch name for a given commit indicator.\n" +
			"Useful for your own custom scripting.",
		Args: cobra.MaximumNArgs(1),
		Annotations: map[string]string{
			"defaultLogLevel": "error",
		},
	}
	indicatorTypeString := addIndicatorFlag(cmd)
	cmd.Run = func(cmd *cobra.Command, args []string) {
		selectCommitOptions := interactive.CommitSelectionOptions{
			Prompt:      "What commit do you want the branch name for?",
			CommitType:  interactive.CommitTypeBoth,
			MultiSelect: false,
		}
		targetCommit := getTargetCommits(appConfig, args, indicatorTypeString, selectCommitOptions)
		util.Fprint(appConfig.Io.Out, targetCommit[0].Branch)
	}
	return cmd
}
