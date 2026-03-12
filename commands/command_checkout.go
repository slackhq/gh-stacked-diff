package commands

import (
	"flag"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func createCheckoutCommand() Command {
	flagSet := flag.NewFlagSet("checkout", flag.ContinueOnError)
	indicatorTypeString := addIndicatorFlag(flagSet)
	return Command{
		FlagSet: flagSet,
		Summary: "Checks out branch associated with commit indicator",
		Description: "Checks out the branch associated with commit indicator.\n" +
			"\n" +
			"For when you want to merge only the branch with with origin/" + util.GetMainBranchForHelp() + ",\n" +
			"rather than your entire local " + util.GetMainBranchForHelp() + " branch, verify why \n" +
			"CI is failing on that particular branch, or for any other reason.\n" +
			"\n" +
			"After modifying the branch you can use \"sd replace-commit\" to sync local " + util.GetMainBranchForHelp() + ".",
		Usage: "sd " + flagSet.Name() + " [flags] <commitIndicator>",
		OnSelected: func(appConfig util.AppConfig, command Command) {
			if flagSet.NArg() > 1 {
				commandError(appConfig, flagSet, "too many arguments", command.Usage)
			}
			selectCommitOptions := interactive.CommitSelectionOptions{
				Prompt:      "What commit do you want to checkout the associated branch for?",
				CommitType:  interactive.CommitTypePr,
				MultiSelect: false,
			}
			targetCommit := getTargetCommits(appConfig, command, []string{flagSet.Arg(0)}, indicatorTypeString, selectCommitOptions)
			util.ExecuteOrDie(
				util.ExecuteOptions{Io: appConfig.Io},
				"git", "checkout", targetCommit[0].Branch,
			)
		}}
}
