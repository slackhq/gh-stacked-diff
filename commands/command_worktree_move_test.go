package commands

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestWorktreeMove_CherryPicksToMainWorktree(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelDebug)

	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetCurrentBranchName())

	mainPath := testutil.SetupSecondaryWorktree(t)

	// Add a commit in the secondary worktree.
	testutil.AddCommit("worktree-commit", "worktree-file")
	commitHash := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "-n", "1", "--pretty=format:%h")

	testParseArguments("worktree-move", commitHash)

	// Verify the commit was cherry-picked to the main worktree.
	mainLog := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "-C", mainPath, "log", "--oneline")
	assert.True(strings.Contains(mainLog, "worktree-commit"))
}

func TestWorktreeMove_MultipleCommits(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelDebug)

	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetCurrentBranchName())

	mainPath := testutil.SetupSecondaryWorktree(t)

	testutil.AddCommit("commit-one", "file-one")
	hash1 := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "-n", "1", "--pretty=format:%h")
	testutil.AddCommit("commit-two", "file-two")
	hash2 := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "-n", "1", "--pretty=format:%h")

	testParseArguments("worktree-move", hash1, hash2)

	mainLog := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "-C", mainPath, "log", "--oneline")
	assert.True(strings.Contains(mainLog, "commit-one"))
	assert.True(strings.Contains(mainLog, "commit-two"))
}

func TestWorktreeMove_WhenNotInSecondaryWorktree_Panics(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelDebug)

	testutil.AddCommit("first", "")
	allCommits := templates.GetAllCommits()

	assert.PanicsWithValue(
		"Panicking instead of exiting with code 1",
		func() {
			testParseArguments("worktree-move", allCommits[0].Commit)
		},
	)
}
