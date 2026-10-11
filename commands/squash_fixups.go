package commands

import (
	"fmt"
	"log/slog"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// Rebases the current branch (expected to be local main), marking fixupCommits as
// fixups that are squashed into targetCommit.
func squashFixupsOnCurrentBranch(targetCommit templates.GitLog, fixupCommits []templates.GitLog) {
	slog.Info(fmt.Sprint("Rebasing, marking as fixup ", fixupCommits, " for target ", targetCommit.Commit))
	commitHashes := util.MapSlice(fixupCommits, func(commit templates.GitLog) string {
		return commit.Commit
	})
	environmentVariables := []string{
		sequenceEditorEnvVar("sequence-editor-mark-as-fixup", append([]string{targetCommit.Commit}, commitHashes...)...),
	}
	slog.Debug(fmt.Sprint("Using sequence editor ", environmentVariables))
	options := util.ExecuteOptions{EnvironmentVariables: environmentVariables, Io: util.GetAppConfig().Io}
	rebaseBase := templates.SortEarliestFirst(append([]templates.GitLog{targetCommit}, fixupCommits...), "")[0].Commit
	gitutil.RebaseAndSkipAllEmptyOrDie(options, "-i", rebaseBase+"^")
}
