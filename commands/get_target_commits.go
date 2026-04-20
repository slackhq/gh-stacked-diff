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
	// Note: empty values are ignored for convienience to allow use of args.
	commitsFromCommandLine []string,
	indicatorTypeString *string,
	options interactive.CommitSelectionOptions,
) []templates.GitLog {
	appConfig := util.GetAppConfig()
	commitsFromCommandLine = util.FilterSlice(commitsFromCommandLine, func(commit string) bool {
		return commit != ""
	})
	if len(commitsFromCommandLine) == 0 {
		messageCannotAskPrefix := "Target commit not specified and cannot ask interactively because "
		if !interactive.InteractiveEnabled() {
			panic(messageCannotAskPrefix + "not a terminal")
		}
		selectedCommits, err := interactive.GetCommitSelection(options)
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
			return templates.ResolveCommitIndicator(commit, indicatorType)
		})
	}
}
