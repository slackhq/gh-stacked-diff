package commands

import (
	"flag"
	"log/slog"
	"strings"

	"github.com/tinyspeck/gh-stacked-diff/v2/util"
)

func createVersionCommand() Command {
	flagSet := flag.NewFlagSet("version", flag.ContinueOnError)
	return Command{
		FlagSet:         flagSet,
		DefaultLogLevel: slog.LevelError,
		Summary:         "Outputs version number",
		Description:     "Outputs the version number.",
		Usage:           "sd " + flagSet.Name(),
		SkipRepoCheck:   true,
		OnSelected: func(appConfig util.AppConfig, command Command) {
			if flagSet.NArg() != 0 {
				commandError(appConfig, flagSet, "too many args", command.Usage)
			}
			var stableSuffix string
			if strings.TrimSpace(util.CurrentVersion) == strings.TrimSpace(util.StableVersion) {
				stableSuffix = " (stable)"
			} else {
				stableSuffix = " (preview)"
			}
			util.Fprintln(appConfig.Io.Out, "Version "+util.CurrentVersion+stableSuffix)
		}}
}
