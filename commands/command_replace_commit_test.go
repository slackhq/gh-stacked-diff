package commands

import (
	"log/slog"
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func TestSdReplaceCommit_WithMultipleCommits_ReplacesCommitWithBranch(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "1")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())
	testutil.AddCommit("second", "will-be-replaced")
	testParseArguments("new", "1")
	testutil.AddCommit("fifth", "5")

	allCommits := templates.GetAllCommits()

	testParseArguments("checkout", allCommits[1].Commit)

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allCommits[2].Commit)
	testutil.AddCommit("on-second-branch-only", "2")
	testutil.AddCommit("on-second-branch-only", "3")
	testutil.AddCommit("on-second-branch-only", "4")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", util.GetMainBranchOrDie())

	testParseArguments("replace-commit", allCommits[1].Commit)

	allCommits = templates.GetAllCommits()

	assert.Equal(4, len(allCommits))
	assert.Equal("fifth", allCommits[0].Subject)
	assert.Equal("second", allCommits[1].Subject)
	assert.Equal("first", allCommits[2].Subject)
	assert.Equal(testutil.InitialCommitSubject, allCommits[3].Subject)

	dirEntries, err := os.ReadDir(".")
	if err != nil {
		panic("Could not read dir: " + err.Error())
	}
	assert.Equal(6, len(dirEntries))
	assert.Equal(".git", dirEntries[0].Name())
	assert.Equal("1", dirEntries[1].Name())
	assert.Equal("2", dirEntries[2].Name())
	assert.Equal("3", dirEntries[3].Name())
	assert.Equal("4", dirEntries[4].Name())
	assert.Equal("5", dirEntries[5].Name())
}

func TestSdReplaceCommit_WhenCherryPickFails_RestoresBranch(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "1")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())
	testutil.AddCommit("second", "will-be-replaced")
	testParseArguments("new", "1")
	// Create a commit after "second" that modifies the same file, so that
	// cherry-picking it back on top of the replaced commit will conflict.
	testutil.CommitFileChange("third", "will-be-replaced", "conflicting content")

	allCommits := templates.GetAllCommits()
	restoreBranch := util.GetCurrentBranchName()

	// On the branch for "second", make a change to the same file so it conflicts.
	testParseArguments("checkout", allCommits[1].Commit)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allCommits[2].Commit)
	testutil.CommitFileChange("on-branch-only", "will-be-replaced", "branch content")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", util.GetMainBranchOrDie())

	defer func() {
		r := recover()
		if r != nil {
			assert.Equal(restoreBranch, util.GetCurrentBranchName())
			assert.Equal(allCommits, templates.GetAllCommits())
		}
	}()

	testParseArguments("replace-commit", "--on-cherry-pick-error=rollback", allCommits[1].Commit)

	assert.Fail("did not panic on cherry-pick conflict")
}

func TestSdReplaceCommit_WhenCherryPickFailsWithPrompt_RestoresBranch(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "1")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())
	testutil.AddCommit("second", "will-be-replaced")
	testParseArguments("new", "1")
	testutil.CommitFileChange("third", "will-be-replaced", "conflicting content")

	allCommits := templates.GetAllCommits()
	restoreBranch := util.GetCurrentBranchName()

	testParseArguments("checkout", allCommits[1].Commit)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allCommits[2].Commit)
	testutil.CommitFileChange("on-branch-only", "will-be-replaced", "branch content")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", util.GetMainBranchOrDie())

	// Confirm rollback (Enter = default Y)
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))

	defer func() {
		r := recover()
		if r != nil {
			assert.Equal(restoreBranch, util.GetCurrentBranchName())
			assert.Equal(allCommits, templates.GetAllCommits())
		}
	}()

	testParseArguments("replace-commit", allCommits[1].Commit)

	assert.Fail("did not panic on cherry-pick conflict")
}

func TestSdReplaceCommit_WhenCherryPickFails_AndExitFlag_DoesNotRestore(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "1")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())
	testutil.AddCommit("second", "will-be-replaced")
	testParseArguments("new", "1")
	// Create a commit after "second" that modifies the same file, so that
	// cherry-picking it back on top of the replaced commit will conflict.
	testutil.CommitFileChange("third", "will-be-replaced", "conflicting content")

	allCommits := templates.GetAllCommits()

	// On the branch for "second", make a change to the same file so it conflicts.
	testParseArguments("checkout", allCommits[1].Commit)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allCommits[2].Commit)
	testutil.CommitFileChange("on-branch-only", "will-be-replaced", "branch content")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", util.GetMainBranchOrDie())

	defer func() {
		r := recover()
		if r != nil {
			// State should NOT be rolled back - commits should differ from original.
			currentCommits := templates.GetAllCommits()
			assert.NotEqual(allCommits, currentCommits)
		}
	}()

	testParseArguments("replace-commit", "--on-cherry-pick-error=exit", allCommits[1].Commit)

	assert.Fail("did not panic on cherry-pick conflict")
}

func TestSdReplaceCommit_WhenCherryPickFails_AndPromptDeclined_DoesNotRestore(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "1")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())
	testutil.AddCommit("second", "will-be-replaced")
	testParseArguments("new", "1")
	testutil.CommitFileChange("third", "will-be-replaced", "conflicting content")

	allCommits := templates.GetAllCommits()

	testParseArguments("checkout", allCommits[1].Commit)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allCommits[2].Commit)
	testutil.CommitFileChange("on-branch-only", "will-be-replaced", "branch content")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", util.GetMainBranchOrDie())

	// Decline rollback (n = manual fix)
	interactive.SendToProgram(0, interactive.NewMessageRune('n'))

	defer func() {
		r := recover()
		if r != nil {
			// State should NOT be rolled back - commits should differ from original.
			currentCommits := templates.GetAllCommits()
			assert.NotEqual(allCommits, currentCommits)
		}
	}()

	testParseArguments("replace-commit", allCommits[1].Commit)

	assert.Fail("did not panic on cherry-pick conflict")
}
