package commands

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	"github.com/fatih/color"
	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"

	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestSdLog_WhenRemoteHasSomeCommits_PrintsNewLogsOnly(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	testutil.AddCommit("second", "")

	out := testParseArguments("log")

	assert.NotContains(out, "first")
	assert.Contains(out, "second")
}

func TestSdLog_WhenPrCreatedForSomeCommits_PrintsCheckForCommitsWithPrs(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	out := testParseArguments("log")

	assert.Contains(out, "✅")
}

func TestSdLog_WhenNotOnMain_OnlyShowsCommitsNotOnMain(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "my-branch")

	testutil.AddCommit("second", "")

	out := testParseArguments("log")

	assert.NotContains(out, "first")
	assert.Contains(out, "second")
}

func TestSdLog_WhenCommitHasBranch_PrintsExtraBranchCommits(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")
	testutil.AddCommit("forth", "")

	testParseArguments("update", "4", "3", "2", "1")

	out := testParseArguments("log")

	allCommits := templates.GetAllCommits()

	assert.Equal("1. ✅ "+color.YellowString(allCommits[0].Commit)+" first\n"+
		"      - second\n"+
		"      - third\n"+
		"      - forth\n",
		out)
}

func TestSdLog_WhenBranchHasManyCommits_PrintsLatest3(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")
	testutil.AddCommit("forth", "")
	testutil.AddCommit("fifth", "")
	testutil.AddCommit("sixth", "")

	testParseArguments("update", "6", "5", "4", "3", "2", "1")

	out := testParseArguments("log")

	allCommits := templates.GetAllCommits()
	assert.Equal("1. ✅ "+color.YellowString(allCommits[0].Commit)+" first\n"+
		"      - [hiding 3 previous...]\n"+
		"      - fifth\n"+
		"      - sixth\n",
		out)
}

func TestSdLog_LogsOutput(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	out := testParseArguments("log")

	assert.Contains(out, "first")
}

func TestSdLog_WhenManyCommits_PadsFirstCommits(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")
	testutil.AddCommit("fourth", "")
	testutil.AddCommit("fifth", "")
	testutil.AddCommit("sixth", "")
	testutil.AddCommit("seventh", "")
	testutil.AddCommit("eigth", "")
	testutil.AddCommit("ninth", "")
	testutil.AddCommit("tenth", "")

	out := testParseArguments("log")

	assert.Contains(out, "\n 2.    ")
	assert.Contains(out, "\n10.    ")
}

func TestSdLog_WhenMultiplePrs_MatchesAllPrs(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")

	testParseArguments("new", "2")
	testParseArguments("new", "1")

	out := testParseArguments("log")

	assert.Regexp("✅.*first", out)
	assert.Regexp("✅.*second", out)
}

func TestSdLog_WhenPollFlag_PollsAndQuitsOnInput(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse(
		"check,COMPLETED,SUCCESS,SUCCESS\nstate,OPEN\nreviewRequestCount,0\nmergeStateStatus,CLEAN\nisDraft,false",
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	out := util.NewWriteRecorder(new(bytes.Buffer))
	done := make(chan struct{})
	go func() {
		testParseArgumentsWithOut(out, "log", "--poll", "--config", "pollInterval=10m")
		close(done)
	}()

	testutil.WaitForOutput(t, out, "[open]")
	assert.Contains(out.String(), "first")

	interactive.SendToProgram(0, interactive.NewMessageRune('q'))
	testutil.WaitForDone(t, done)
}

func TestSdLog_WhenStatusFlagNotOnMain_Panics(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "my-branch")

	out := new(bytes.Buffer)
	defer func() {
		r := recover()
		assert.NotNil(r)
		assert.Contains(out.String(), "--status is only supported on the main branch")
	}()
	testParseArgumentsWithOut(out, "log", "--status")
	assert.Fail("did not panic")
}

func TestSdLog_WhenStatusFlag_ShowsStatusInfo(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse(
		"check,COMPLETED,SUCCESS,SUCCESS\nstate,OPEN\nreviewRequestCount,1\nlatestReview,someuser,APPROVED,4,0\nmergeStateStatus,CLEAN\nisDraft,false",
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.Contains(out, "first")
	assert.Contains(out, "[open]")
	assert.Contains(out, "[checks: passed")
	assert.Contains(out, "[approved: 1/2]")
	assert.Contains(out, "[can merge]")
}

func TestSdLog_WhenStatusFlag_ShowsChangesRequested(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse(
		"check,COMPLETED,SUCCESS,SUCCESS\nstate,OPEN\nreviewRequestCount,1\nlatestReview,alice,CHANGES_REQUESTED,0,0\nlatestReview,bob,APPROVED,50,0\nmergeStateStatus,BLOCKED\nisDraft,false",
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.Contains(out, "[approved: 0/3]")
	assert.Contains(out, "alice requested changes")
	assert.Contains(out, "bob approved with comments")
	assert.NotContains(out, "[can merge]")
}

func TestSdLog_WhenStatusFlag_CombinesUsersWithSameStatus(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse(
		"check,COMPLETED,SUCCESS,SUCCESS\nstate,OPEN\nreviewRequestCount,0\nlatestReview,alice,CHANGES_REQUESTED,0,0\nlatestReview,bob,CHANGES_REQUESTED,0,0\nlatestReview,carol,APPROVED,0,0\nlatestReview,dave,APPROVED,0,0\nmergeStateStatus,BLOCKED\nisDraft,false",
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.Contains(out, "alice, bob requested changes")
	assert.Contains(out, "carol, dave approved")
	assert.NotContains(out, "alice requested changes\n")
	assert.NotContains(out, "bob requested changes\n")
}

func TestSdLog_WhenStatusFlag_ShowsMergedStatus(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse(
		"check,COMPLETED,SUCCESS,SUCCESS\nstate,MERGED\nreviewRequestCount,0\nlatestReview,someuser,APPROVED,4,0\nmergeStateStatus,CLEAN\nisDraft,false",
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.Contains(out, "[merged]")
	assert.Contains(out, "[checks: passed")
}

func TestSdLog_WhenStatusFlagAndNoPRs_PrintsCommitsWithoutBubbletea(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")

	out := testParseArguments("log", "--status")

	assert.Contains(out, "first")
	assert.Contains(out, "second")
	assert.NotContains(out, "✅")
}

func TestSdLog_WhenStatusFlag_ShowsDraftStatus(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse(
		"state,OPEN\nreviewRequestCount,0\nmergeStateStatus,BLOCKED\nisDraft,true",
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.NotContains(out, "[open]")
	assert.Contains(out, "[draft]")
}

func TestSdLog_WhenOtherWorktreeHasUniqueCommits_ShowsWorktreeCommits(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("main-commit", "")

	mainPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Create a secondary worktree with its own commit.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "secondary-branch")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../secondary-worktree", "secondary-branch")
	t.Cleanup(func() {
		_ = os.Chdir(mainPath)
		_, _ = util.Execute(util.ExecuteOptions{}, "git", "worktree", "remove", "--force", "../secondary-worktree")
	})

	if err := os.Chdir("../secondary-worktree"); err != nil {
		t.Fatal(err)
	}
	testutil.AddCommit("secondary-commit", "secondary-file")

	// Go back to main worktree.
	if err := os.Chdir(mainPath); err != nil {
		t.Fatal(err)
	}
	gitutil.ResetCacheForTesting()

	out := testParseArguments("log")

	assert.Contains(out, "main-commit")
	assert.Contains(out, "secondary-commit")
	assert.Contains(out, "secondary-worktree (secondary-branch)")
}

func TestSdLog_WhenStatusFlagAndOtherWorktree_ShowsWorktreeCommits(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("main-commit", "")
	testParseArguments("new", "1")

	mainPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Create a secondary worktree with its own commit.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "secondary-branch")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../secondary-worktree", "secondary-branch")
	t.Cleanup(func() {
		_ = os.Chdir(mainPath)
		_, _ = util.Execute(util.ExecuteOptions{}, "git", "worktree", "remove", "--force", "../secondary-worktree")
	})

	if err := os.Chdir("../secondary-worktree"); err != nil {
		t.Fatal(err)
	}
	testutil.AddCommit("secondary-commit", "secondary-file")

	// Go back to main worktree.
	if err := os.Chdir(mainPath); err != nil {
		t.Fatal(err)
	}
	gitutil.ResetCacheForTesting()

	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse(
		"check,COMPLETED,SUCCESS,SUCCESS\nstate,OPEN\nreviewRequestCount,0\nmergeStateStatus,CLEAN\nisDraft,false",
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.Contains(out, "main-commit")
	assert.Contains(out, "secondary-commit")
	assert.Contains(out, "secondary-worktree (secondary-branch)")
}

func TestSdLog_WhenOtherWorktreeHasOnlySharedCommits_DoesNotShowWorktreeSection(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("shared-commit", "")

	mainPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Create a secondary worktree that has the same commits as main (no unique ones).
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "secondary-branch")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../secondary-worktree", "secondary-branch")
	t.Cleanup(func() {
		_ = os.Chdir(mainPath)
		_, _ = util.Execute(util.ExecuteOptions{}, "git", "worktree", "remove", "--force", "../secondary-worktree")
	})

	gitutil.ResetCacheForTesting()

	out := testParseArguments("log")

	assert.Contains(out, "shared-commit")
	assert.NotContains(out, "secondary-worktree")
}
