package commands

import (
	"log/slog"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func TestSdUpdateBranches_DraftBranch_RecreatesFromOriginMain(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Add "first" commit and create PR branch
	testutil.CommitFileChange("first", "file1", "original")
	testParseArguments("new", "1")

	// Amend the commit on main so its diff differs from the branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--soft", "HEAD~1")
	testutil.CommitFileChange("first", "file1", "amended")

	allCommits := templates.GetAllCommits()

	// Mock PR status as draft
	testExecutor.SetResponse("isDraft,true\nstate,OPEN\nnumber,1\nreviewRequestCount,0\nmergeStateStatus,BLOCKED",
		nil, "gh", "pr", "view", allCommits[0].Branch, util.MatchAnyRemainingArgs)

	// Select the commit in the update dialog (enter selects current row and confirms)
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))

	testParseArguments("update-branches")

	// Verify branch was updated with the amended commit
	branchFileContent := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "show", allCommits[0].Branch+":file1")
	assert.Equal("amended", branchFileContent)
}

func TestSdUpdateBranches_NonDraftBranch_MergesOriginMain(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Push "other-change" to origin so origin/main has content the branch won't have
	testutil.AddCommit("other-change", "other-file")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", "HEAD~1")

	// Add "first" commit and create PR branch (based on initial commit, before other-change)
	testutil.CommitFileChange("first", "file1", "original")
	testParseArguments("new", "1")

	// Amend the commit on main so its diff differs from the branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--soft", "HEAD~1")
	testutil.CommitFileChange("first", "file1", "amended")

	allCommits := templates.GetAllCommits()

	// Mock PR status as NOT draft
	testExecutor.SetResponse("isDraft,false\nstate,OPEN\nnumber,1\nreviewRequestCount,0\nmergeStateStatus,CLEAN",
		nil, "gh", "pr", "view", allCommits[0].Branch, util.MatchAnyRemainingArgs)

	// Select the commit in the update dialog
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))

	testParseArguments("update-branches")

	// Verify branch has a merge commit
	branchLog := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "log", "--format=%s", allCommits[0].Branch)
	assert.Contains(branchLog, "Merge")
}

func TestSdUpdateBranches_BranchAlreadyInSync_SkipsDialog(t *testing.T) {
	assert := assert.New(t)
	_ = testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "file1")
	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()

	// No SendToProgram needed — dialog should not appear since diffs match

	testParseArguments("update-branches")

	// Verify branch still exists unchanged
	branches := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch")
	assert.Contains(branches, allCommits[0].Branch)
}

func TestSdUpdateBranches_UserCancels_NoBranchUpdate(t *testing.T) {
	assert := assert.New(t)
	_ = testutil.InitTest(t, slog.LevelError)

	// Add "first" commit and create PR branch
	testutil.CommitFileChange("first", "file1", "original")
	testParseArguments("new", "1")

	// Amend the commit on main so its diff differs from the branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--soft", "HEAD~1")
	testutil.CommitFileChange("first", "file1", "amended")

	allCommits := templates.GetAllCommits()
	branchLogBefore := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "log", "--format=%H", allCommits[0].Branch)

	// User cancels the dialog
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEsc))

	testParseArguments("update-branches")

	// Verify branch is unchanged
	branchLogAfter := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "log", "--format=%H", allCommits[0].Branch)
	assert.Equal(branchLogBefore, branchLogAfter)
}
