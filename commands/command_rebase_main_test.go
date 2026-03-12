package commands

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func TestSdRebaseMain_WithDifferentCommits_DropsCommits(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "rebase-will-keep-this-file")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())

	allOriginalCommits := templates.GetAllCommits()

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allOriginalCommits[1].Commit)

	testutil.AddCommit("second", "rebase-will-drop-this-file")

	// Mock closed PRs response (empty - no closed PRs)
	testExecutor.SetResponse("",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "closed", "--search", "is:unmerged", util.MatchAnyRemainingArgs)

	// Mock merged PRs response
	testExecutor.SetResponse(allOriginalCommits[0].Branch+" fakeMergeCommit",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "merged", util.MatchAnyRemainingArgs)

	testParseArguments("rebase-main")

	dirEntries, err := os.ReadDir(".")
	if err != nil {
		panic("Could not read dir: " + err.Error())
	}
	assert.Equal(3, len(dirEntries))
	assert.Equal(".git", dirEntries[0].Name())
	assert.Equal("first", dirEntries[1].Name())
	assert.Equal("rebase-will-keep-this-file", dirEntries[2].Name())
}

func TestSdRebaseMain_WithMulitpleMergedBranches_DropsCommits(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "1")
	testutil.AddCommit("second", "2")
	testutil.AddCommit("third", "3")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())

	allOriginalCommits := templates.GetAllCommits()

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allOriginalCommits[2].Commit)

	testutil.AddCommit("second", "2-rebase-will-drop-this-file")
	testutil.AddCommit("third", "3-rebase-will-drop-this-file")
	testutil.AddCommit("fourth", "4")

	// Mock closed PRs response (empty - no closed PRs)
	testExecutor.SetResponse("",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "closed", "--search", "is:unmerged", util.MatchAnyRemainingArgs)

	// Mock merged PRs response
	testExecutor.SetResponse(
		allOriginalCommits[0].Branch+" fakeMergeCommit\n"+
			allOriginalCommits[1].Branch+" fakeMergeCommit",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "merged", util.MatchAnyRemainingArgs)

	testParseArguments("rebase-main")

	dirEntries, err := os.ReadDir(".")
	if err != nil {
		panic("Could not read dir: " + err.Error())
	}
	assert.Equal(5, len(dirEntries))
	assert.Equal(".git", dirEntries[0].Name())
	assert.Equal("1", dirEntries[1].Name())
	assert.Equal("2", dirEntries[2].Name())
	assert.Equal("3", dirEntries[3].Name())
	assert.Equal("4", dirEntries[4].Name())
}

func TestSdRebaseMain_WithDuplicateBranches_Panics(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "1")
	testutil.AddCommit("second", "2.1")
	testutil.AddCommit("second", "2.2")

	allOriginalCommits := templates.GetAllCommits()

	// Mock closed PRs response (empty - no closed PRs)
	testExecutor.SetResponse("",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "closed", "--search", "is:unmerged", util.MatchAnyRemainingArgs)

	// Mock merged PRs response
	testExecutor.SetResponse(allOriginalCommits[0].Branch+" fakeMergeCommit",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "merged", util.MatchAnyRemainingArgs)

	// Return on panic
	defer func() { _ = recover() }()

	testParseArguments("rebase-main")

	assert.Fail("did not panic with duplicate branches")
}

func TestSdRebaseMain_WhenRebaseFails_DropsBranches(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "file-with-conflicts")
	testutil.CommitFileChange("second", "change-value-to-avoid-same-hash", "1")
	testutil.CommitFileChange("third", "file-with-conflicts", "1")
	testParseArguments("new", "2")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())

	allCommits := templates.GetAllCommits()
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allCommits[2].Commit)
	// If this runs in the same second then it will generate the same commit hash, so change value.
	testutil.CommitFileChange("second", "change-value-to-avoid-same-hash", "2")
	testutil.CommitFileChange("fourth", "file-with-conflicts", "2")

	// Mock closed PRs response (empty - no closed PRs)
	testExecutor.SetResponse("",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "closed", "--search", "is:unmerged", util.MatchAnyRemainingArgs)

	// Mock merged PRs response
	testExecutor.SetResponse(allCommits[1].Branch+" fakeMergeCommit",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "merged", util.MatchAnyRemainingArgs)

	branches := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch")
	assert.Contains(branches, "second")

	out := new(bytes.Buffer)
	defer func() {
		r := recover()
		if r != nil {
			assert.Contains(out.String(), "Rebase failed")
			branches = util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch")
			assert.NotContains(branches, "second")
		}
	}()

	testParseArgumentsWithOut(out, "--log-level=info", "rebase-main")

	assert.Fail("did not panic with rebase fail")
}

func TestSdRebaseMain_WithMergedPrAlreadyRebased_KeepsCommits(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "second-1")
	testutil.AddCommit("third", "")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())
	allCommits := templates.GetAllCommits()
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allCommits[1].Commit)

	testutil.AddCommit("second", "second-2")
	testParseArguments("new", "1")

	// Use the commit of the first "second" commit as the branch
	// that was merged so that the second "second" commit is not dropped.
	// Mock closed PRs response (empty - no closed PRs)
	testExecutor.SetResponse("",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "closed", "--search", "is:unmerged", util.MatchAnyRemainingArgs)

	// Mock merged PRs response
	testExecutor.SetResponse(allCommits[1].Branch+" "+allCommits[1].Commit,
		nil, "gh", "pr", "list", "--author", "@me", "--state", "merged", util.MatchAnyRemainingArgs)

	testParseArguments("rebase-main")

	branches := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch")
	assert.Contains(branches, "second")
}

func TestSdRebaseMain_WithDroppedCommits_DropsBranches(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "rebase-will-keep-this-file")

	testParseArguments("new", "1")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())

	allOriginalCommits := templates.GetAllCommits()

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allOriginalCommits[1].Commit)

	testutil.AddCommit("second", "rebase-will-drop-this-file")

	// Mock closed PRs response (empty - no closed PRs)
	testExecutor.SetResponse("",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "closed", "--search", "is:unmerged", util.MatchAnyRemainingArgs)

	// Mock merged PRs response
	testExecutor.SetResponse(allOriginalCommits[0].Branch+" fakeMergeCommit",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "merged", util.MatchAnyRemainingArgs)

	testParseArguments("rebase-main")

	assert.False(util.RemoteHasBranch(allOriginalCommits[0].Branch))
	assert.False(util.GetLocalHasBranchOrDie(allOriginalCommits[0].Branch))
}

func TestSdRebaseMain_WithClosedPRConfirmed_DropsCommits(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "rebase-will-keep-this-file")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())

	allOriginalCommits := templates.GetAllCommits()

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allOriginalCommits[1].Commit)

	testutil.AddCommit("second", "rebase-will-drop-this-file")

	newCommits := templates.GetAllCommits()

	// Mock closed PRs response
	testExecutor.SetResponse(allOriginalCommits[0].Branch+" "+newCommits[0].Commit,
		nil, "gh", "pr", "list", "--author", "@me", "--state", "closed", "--search", "is:unmerged", util.MatchAnyRemainingArgs)

	// Mock merged PRs response (empty - no merged PRs)
	testExecutor.SetResponse("",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "merged", util.MatchAnyRemainingArgs)

	// Simulate user confirming (y) to drop closed PR commits
	interactive.SendToProgram(0, interactive.NewMessageRune('y'))

	testParseArguments("rebase-main")

	dirEntries, err := os.ReadDir(".")
	if err != nil {
		panic("Could not read dir: " + err.Error())
	}
	assert.Equal(3, len(dirEntries))
	assert.Equal(".git", dirEntries[0].Name())
	assert.Equal("first", dirEntries[1].Name())
	assert.Equal("rebase-will-keep-this-file", dirEntries[2].Name())
}

func TestSdRebaseMain_WithClosedPRDeclined_KeepsCommits(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "file2")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())

	allOriginalCommits := templates.GetAllCommits()

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allOriginalCommits[1].Commit)

	testutil.AddCommit("second", "file3")

	newCommits := templates.GetAllCommits()

	// Mock closed PRs response
	testExecutor.SetResponse(allOriginalCommits[0].Branch+" "+newCommits[0].Commit,
		nil, "gh", "pr", "list", "--author", "@me", "--state", "closed", "--search", "is:unmerged", util.MatchAnyRemainingArgs)

	// Mock merged PRs response (empty - no merged PRs)
	testExecutor.SetResponse("",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "merged", util.MatchAnyRemainingArgs)

	// Simulate user declining (n) to drop closed PR commits
	interactive.SendToProgram(0, interactive.NewMessageRune('n'))

	testParseArguments("rebase-main")

	dirEntries, err := os.ReadDir(".")
	if err != nil {
		panic("Could not read dir: " + err.Error())
	}
	// Should keep the "second" commit since user declined
	// After rebasing onto origin/main (which has file2), the new "second" commit (with file3) is kept
	assert.Equal(4, len(dirEntries))
	assert.Equal(".git", dirEntries[0].Name())
	assert.Equal("file2", dirEntries[1].Name())
	assert.Equal("file3", dirEntries[2].Name())
	assert.Equal("first", dirEntries[3].Name())
}

func TestSdRebaseMain_WithMergedAndClosedPRs_DropsBothWhenConfirmed(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "1")
	testutil.AddCommit("second", "2")
	testutil.AddCommit("third", "3")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())

	allOriginalCommits := templates.GetAllCommits()

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allOriginalCommits[2].Commit)

	testutil.AddCommit("second", "2-rebase-will-drop-this-file")
	testutil.AddCommit("third", "3-rebase-will-drop-this-file")

	newCommits := templates.GetAllCommits()

	// Mock closed PRs response - "third" is closed
	testExecutor.SetResponse(allOriginalCommits[0].Branch+" "+newCommits[0].Commit,
		nil, "gh", "pr", "list", "--author", "@me", "--state", "closed", "--search", "is:unmerged", util.MatchAnyRemainingArgs)

	// Mock merged PRs response - "second" is merged
	testExecutor.SetResponse(allOriginalCommits[1].Branch+" fakeMergeCommit",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "merged", util.MatchAnyRemainingArgs)

	// Simulate user confirming (y) to drop closed PR commits
	interactive.SendToProgram(0, interactive.NewMessageRune('y'))

	testParseArguments("rebase-main")

	dirEntries, err := os.ReadDir(".")
	if err != nil {
		panic("Could not read dir: " + err.Error())
	}
	// Should drop both "second" (merged) and "third" (closed)
	// After rebasing onto origin/main which has all 3 original files (1, 2, 3)
	assert.Equal(4, len(dirEntries))
	assert.Equal(".git", dirEntries[0].Name())
	assert.Equal("1", dirEntries[1].Name())
	assert.Equal("2", dirEntries[2].Name())
	assert.Equal("3", dirEntries[3].Name())
}

func TestSdRebaseMain_WhenCommitAlreadyCherryPickedToBranch_SkipsEmptyCommit(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelDebug)

	util.ExecuteOrDie(util.ExecuteOptions{}, "touch", "second")
	testutil.AddCommit("first", "")
	testParseArguments("new", "1")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", "HEAD~1")
	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")

	allCommits := templates.GetAllCommits()

	testExecutor.SetResponse(allCommits[1].Branch+" fakeMergeCommit",
		nil, "gh", "pr", "list", "--author", "@me", "--state", "merged", util.MatchAnyRemainingArgs)

	testParseArguments("rebase-main")

	afterRebaseCommits := templates.GetAllCommits()
	assert.Equal(2, len(afterRebaseCommits))
	assert.Equal(allCommits[1].Subject, afterRebaseCommits[0].Subject)
	assert.Equal(testutil.InitialCommitSubject, afterRebaseCommits[1].Subject)
}
