package commands

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// Guaranteed to return at least one value (or else appConfig.Exit will be called).
//
// Returned commits are ordered earliest to latest. commitsFromCommandLine can be in any order.
func getTargetCommits(
	commitsFromCommandLine []string,
	indicatorTypeString *string,
	options interactive.CommitSelectionOptions,
) []templates.GitLog {
	return sortEarliestFirst(getUnsortedTargetCommits(commitsFromCommandLine, indicatorTypeString, options), options.GitDir)
}

// Sorts commits so that the earliest commit is first, so that they can be cherry-picked in order.
func sortEarliestFirst(commits []templates.GitLog, gitDir string) []templates.GitLog {
	// New commits are ordered newest first, so a higher index is an earlier commit.
	newCommits := templates.GetNewCommits("HEAD", gitDir)
	indexOf := func(commit templates.GitLog) int {
		return slices.IndexFunc(newCommits, func(newCommit templates.GitLog) bool {
			return newCommit.Commit == commit.Commit
		})
	}
	sorted := slices.Clone(commits)
	slices.SortStableFunc(sorted, func(a, b templates.GitLog) int {
		return indexOf(b) - indexOf(a)
	})
	return sorted
}

func getUnsortedTargetCommits(
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
