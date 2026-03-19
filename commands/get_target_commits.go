package commands

import (
	"fmt"
	"log/slog"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// Guaranteed to return at least one value (or else appConfig.Exit will be called).
func getTargetCommits(
	appConfig util.AppConfig,
	// Note: empty values are ignored for convienience to allow use of args.
	commitsFromCommandLine []string,
	indicatorTypeString *string,
	options interactive.CommitSelectionOptions,
) []templates.GitLog {
	commitsFromCommandLine = util.FilterSlice(commitsFromCommandLine, func(commit string) bool {
		return commit != ""
	})
	if len(commitsFromCommandLine) == 0 {
		messageCannotAskPrefix := "Target commit not specified and cannot ask interactively because "
		if !interactive.InteractiveEnabled(appConfig) {
			panic(messageCannotAskPrefix + "not a terminal")
		}
		selectedCommits, err := interactive.GetCommitSelection(appConfig.Io, options)
		if err != nil {
			panic(messageCannotAskPrefix + err.Error())
		}
		if len(selectedCommits) == 0 {
			appConfig.Exit(0)
		}
		slog.Info("Target commits: " + fmt.Sprint(selectedCommits))
		return selectedCommits
	} else {
		indicatorType := checkIndicatorFlag(indicatorTypeString)
		return util.MapSlice(commitsFromCommandLine, func(commit string) templates.GitLog {
			return templates.GetBranchInfo(commit, indicatorType)
		})
	}
}
