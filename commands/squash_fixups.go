package commands

import (
	"fmt"
	"log/slog"
	"slices"

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
	rebaseBase := earliestCommit(targetCommit, fixupCommits)
	gitutil.RebaseAndSkipAllEmptyOrDie(options, "-i", rebaseBase+"^")
}

// earliestCommit returns the commit hash of the oldest commit among destCommit
// and commitsToCherryPick. GetNewCommits returns newest-first, so the highest
// index is the oldest commit.
func earliestCommit(destCommit templates.GitLog, commitsToCherryPick []templates.GitLog) string {
	newCommits := templates.GetNewCommits("HEAD", "")
	earliest := destCommit.Commit
	earliestIdx := slices.IndexFunc(newCommits, func(gl templates.GitLog) bool {
		return gl.Commit == destCommit.Commit
	})
	for _, cp := range commitsToCherryPick {
		idx := slices.IndexFunc(newCommits, func(gl templates.GitLog) bool {
			return gl.Commit == cp.Commit
		})
		if idx > earliestIdx {
			earliestIdx = idx
			earliest = cp.Commit
		}
	}
	return earliest
}
