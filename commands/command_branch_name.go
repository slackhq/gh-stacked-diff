package commands

import (
	"flag"
	"log/slog"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func createBranchNameCommand() Command {
	flagSet := flag.NewFlagSet("branch-name", flag.ContinueOnError)
	indicatorTypeString := addIndicatorFlag(flagSet)
	return Command{
		FlagSet:         flagSet,
		DefaultLogLevel: slog.LevelError,
		Summary:         "Outputs branch name of commit",
		Description: "Outputs the branch name for a given commit indicator.\n" +
			"Useful for your own custom scripting.",
		Usage: "sd " + flagSet.Name() + " [flags] <commitIndicator>",
		OnSelected: func(appConfig util.AppConfig, command Command) {
			if flagSet.NArg() > 1 {
				commandError(appConfig, flagSet, "too many arguments", command.Usage)
			}
			selectCommitOptions := interactive.CommitSelectionOptions{
				Prompt:      "What commit do you want the branch name for?",
				CommitType:  interactive.CommitTypeBoth,
				MultiSelect: false,
			}
			targetCommit := getTargetCommits(appConfig, command, []string{flagSet.Arg(0)}, indicatorTypeString, selectCommitOptions)
			util.Fprint(appConfig.Io.Out, targetCommit[0].Branch)
		}}
}
