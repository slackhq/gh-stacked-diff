package commands

import (
	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createCheckoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkout [commitIndicator]",
		Short: "Checks out branch associated with commit indicator",
		Long: "Checks out the branch associated with commit indicator.\n" +
			"\n" +
			"For when you want to merge only the branch with with origin/" + gitutil.GetMainBranchForHelp() + ",\n" +
			"rather than your entire local " + gitutil.GetMainBranchForHelp() + " branch, verify why \n" +
			"CI is failing on that particular branch, or for any other reason.\n" +
			"\n" +
			"After modifying the branch you can use \"sd replace-commit\" to sync local " + gitutil.GetMainBranchForHelp() + ".",
		Args: cobra.MaximumNArgs(1),
	}
	indicatorTypeString := addIndicatorFlag(cmd)
	cmd.Run = func(cmd *cobra.Command, args []string) {
		appConfig := util.GetAppConfig()
		selectCommitOptions := interactive.CommitSelectionOptions{
			Prompt:      "What commit do you want to checkout the associated branch for?",
			CommitType:  interactive.CommitTypePr,
			MultiSelect: false,
		}
		targetCommit := getTargetCommits(args, indicatorTypeString, selectCommitOptions)
		util.ExecuteOrDie(
			util.ExecuteOptions{Io: appConfig.Io},
			"git", "checkout", targetCommit[0].Branch,
		)
	}
	return cmd
}
