package commands

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func TestWorktreeMove_CherryPicksToMainWorktree(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

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
	testutil.InitTest(t, slog.LevelError)

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

func TestWorktreeMove_WhenNotInSecondaryWorktree_AndNoWorktrees_Panics(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	allCommits := templates.GetAllCommits()

	assert.PanicsWithValue(
		"Panicking instead of exiting with code 1",
		func() {
			testParseArguments("worktree-move", allCommits[0].Commit)
		},
	)
}

func TestWorktreeMove_WhenNotInSecondaryWorktree_WithWorktreeFlag(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetCurrentBranchName())

	mainPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Create a secondary worktree.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "secondary-branch")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../secondary-worktree", "secondary-branch")
	t.Cleanup(func() {
		_ = os.Chdir(mainPath)
		_, _ = util.Execute(util.ExecuteOptions{}, "git", "worktree", "remove", "--force", "../secondary-worktree")
	})

	// Switch to secondary worktree, add a commit, switch back.
	if err := os.Chdir("../secondary-worktree"); err != nil {
		t.Fatal(err)
	}
	testutil.AddCommit("secondary-commit", "secondary-file")
	commitHash := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "-n", "1", "--pretty=format:%h")
	if err := os.Chdir(mainPath); err != nil {
		t.Fatal(err)
	}

	testParseArguments("worktree-move", "-w", "secondary-branch", commitHash)

	// Verify the commit was cherry-picked to the main worktree.
	mainLog := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "--oneline")
	assert.True(strings.Contains(mainLog, "secondary-commit"))
}

func TestWorktreeMove_WhenCherryPickFails_AndRollbackConfirmed_AbortsCherryPick(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "conflict-file")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetCurrentBranchName())

	mainPath := testutil.SetupSecondaryWorktree(t)

	// In secondary worktree, modify the same file to create a conflict.
	testutil.CommitFileChange("secondary-change", "conflict-file", "secondary content")
	commitHash := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "-n", "1", "--pretty=format:%h")

	// In main worktree, also modify the same file so cherry-pick will conflict.
	if err := os.Chdir(mainPath); err != nil {
		t.Fatal(err)
	}
	testutil.CommitFileChange("main-change", "conflict-file", "main content")
	mainCommitsBefore := templates.GetAllCommits()

	// Switch back to secondary worktree.
	if err := os.Chdir("../secondary-worktree"); err != nil {
		t.Fatal(err)
	}

	// Confirm rollback (Enter = default Y).
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))

	defer func() {
		r := recover()
		if r != nil {
			// Verify we get exit code 1 (the error propagates).
			assert.Equal("Panicking instead of exiting with code 1", r)
			// Verify main worktree is not in a cherry-pick state.
			if err := os.Chdir(mainPath); err != nil {
				t.Fatal(err)
			}
			mainCommitsAfter := templates.GetAllCommits()
			assert.Equal(mainCommitsBefore, mainCommitsAfter)
			return
		}
		assert.Fail("did not panic on cherry-pick conflict")
	}()

	testParseArguments("worktree-move", commitHash)
}

func TestWorktreeMove_WhenCherryPickFails_AndRollbackDeclined_PrintsInstructions(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "conflict-file")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetCurrentBranchName())

	mainPath := testutil.SetupSecondaryWorktree(t)

	// In secondary worktree, modify the same file to create a conflict.
	testutil.CommitFileChange("secondary-change", "conflict-file", "secondary content")
	commitHash := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "-n", "1", "--pretty=format:%h")

	// In main worktree, also modify the same file so cherry-pick will conflict.
	if err := os.Chdir(mainPath); err != nil {
		t.Fatal(err)
	}
	testutil.CommitFileChange("main-change", "conflict-file", "main content")

	// Switch back to secondary worktree.
	if err := os.Chdir("../secondary-worktree"); err != nil {
		t.Fatal(err)
	}

	// Decline rollback (n = manual fix).
	interactive.SendToProgram(0, interactive.NewMessageRune('n'))

	defer func() {
		r := recover()
		if r == nil {
			assert.Fail("did not panic on cherry-pick conflict")
		}
		// Verify working directory changed to main worktree.
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(mainPath, cwd)
	}()

	out := testParseArguments("worktree-move", commitHash)
	assert.Contains(out, "Fix the conflicts")
	assert.Contains(out, "cherry-pick --continue")
	assert.Contains(out, "cherry-pick --abort")
}

func TestWorktreeMove_WhenNotInSecondaryWorktree_InvalidWorktreeFlag_Panics(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetCurrentBranchName())

	mainPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Create a secondary worktree but stay in the main worktree.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "secondary-branch")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../secondary-worktree", "secondary-branch")
	t.Cleanup(func() {
		_ = os.Chdir(mainPath)
		_, _ = util.Execute(util.ExecuteOptions{}, "git", "worktree", "remove", "--force", "../secondary-worktree")
	})

	assert.PanicsWithValue(
		"Panicking instead of exiting with code 1",
		func() {
			testParseArguments("worktree-move", "-w", "nonexistent-branch", "abc123")
		},
	)
}
