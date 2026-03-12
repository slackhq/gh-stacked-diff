package commands

import (
	"flag"
	"log/slog"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func createPrsCommand() Command {
	flagSet := flag.NewFlagSet("prs", flag.ContinueOnError)

	return Command{
		FlagSet: flagSet,
		Summary: "Lists all Pull Requests you have open.",
		Description: "Lists all Pull Requests you have open.\n" +
			"\n" +
			"You must be logged-in, via \"gh auth login\"",
		Usage:           "sd " + flagSet.Name(),
		DefaultLogLevel: slog.LevelError,
		OnSelected: func(appConfig util.AppConfig, command Command) {
			if flagSet.NArg() != 0 {
				commandError(appConfig, flagSet, "too many arguments", command.Usage)
			}
			util.ExecuteOrDie(util.ExecuteOptions{Io: appConfig.Io, Retries: util.GhRetries},
				"gh", "pr", "list", "--author", "@me")
		}}
}
