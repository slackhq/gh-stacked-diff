package commands

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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

	assert.Contains(out, "✓")
}

func TestSdLog_WhenBranchNotOnRemote_PrintsRedCheck(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	originalNoColor := color.NoColor
	color.NoColor = false
	t.Cleanup(func() { color.NoColor = originalNoColor })

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	out := testParseArguments("log")
	assert.Contains(out, color.GreenString("✓"))
	assert.NotContains(out, color.RedString("✓"))

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "-dr", "origin/"+templates.GetAllCommits()[0].Branch)

	out = testParseArguments("log")
	assert.Contains(out, color.RedString("✓"))
	assert.NotContains(out, color.GreenString("✓"))
}

func TestSdLog_WhenDuplicateSubjects_PrintsYellowCircle(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("same subject", "file1")
	testutil.AddCommit("same subject", "file2")

	out := testParseArguments("log")

	assert.Contains(out, "●")
	assert.NotContains(out, "✓")
	assert.Contains(out, "indicates that multiple commits have the same subject")
}

func TestSdLog_WhenUniqueSubjects_DoesNotPrintYellowCircle(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")

	out := testParseArguments("log")

	assert.NotContains(out, "●")
	assert.NotContains(out, "indicates that multiple commits have the same subject")
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

	assert.Equal("1. ✓ "+color.YellowString(allCommits[0].Commit)+" first\n"+
		"     - second\n"+
		"     - third\n"+
		"     - forth\n",
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
	assert.Equal("1. ✓ "+color.YellowString(allCommits[0].Commit)+" first\n"+
		"     - [hiding 3 previous...]\n"+
		"     - fifth\n"+
		"     - sixth\n",
		out)
}

func TestSdLog_LogsOutput(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	out := testParseArguments("log")

	assert.Contains(out, "first")
	assert.NotContains(out, "⬇️")
	assert.Contains(out, "⬆️ stack-2")
	assert.Contains(out, "🥞 (1/2)")
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

	assert.Contains(out, "\n 2.   ")
	assert.Contains(out, "\n10.   ")
}

func TestSdLog_WhenMultiplePrs_MatchesAllPrs(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")

	testParseArguments("new", "2")
	testParseArguments("new", "1")

	out := testParseArguments("log")

	assert.Regexp("✓.*first", out)
	assert.Regexp("✓.*second", out)
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
		"check,COMPLETED,SUCCESS,\nstate,OPEN\nreviewRequestCount,0\nmergeStateStatus,CLEAN\nisDraft,false\nautoMerge,false",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

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

func TestSdLog_WhenStatusFlag_BranchCommitsAppearBeforeApiCalls(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create 3 commits with PRs, each with an extra branch commit.
	// maxParallelAPICalls is 2, so the 3rd PR's API call must wait
	// for a semaphore slot. Branch commits should still appear immediately
	// since they are fetched outside the semaphore.
	testutil.AddCommit("first", "")
	testParseArguments("new", "1")
	testutil.AddCommit("first-extra", "")
	testParseArguments("update", "2", "1")

	testutil.AddCommit("second", "")
	testParseArguments("new", "1")
	testutil.AddCommit("second-extra", "")
	testParseArguments("update", "2", "1")

	testutil.AddCommit("third", "")
	testParseArguments("new", "1")
	testutil.AddCommit("third-extra", "")
	testParseArguments("update", "2", "1")

	// Block gh api graphql calls until we release them.
	apiGate := make(chan struct{})
	prResponse := "state,OPEN\nreviewRequestCount,0\nmergeStateStatus,CLEAN\nisDraft,false\nautoMerge,false"
	testExecutor.SetResponseFunc(prResponse, nil, func(programName string, args ...string) bool {
		if programName == "gh" && len(args) >= 2 && args[0] == "api" && args[1] == "graphql" {
			<-apiGate
			return true
		}
		return false
	})

	out := util.NewWriteRecorder(new(bytes.Buffer))
	done := make(chan struct{})
	go func() {
		testParseArgumentsWithOut(out, "log", "--status")
		close(done)
	}()

	// All branch commits should appear even though API calls are blocked,
	// because GetNewCommits runs outside the semaphore.
	testutil.WaitForOutput(t, out, "first-extra")
	testutil.WaitForOutput(t, out, "second-extra")
	testutil.WaitForOutput(t, out, "third-extra")

	// No status info should be present yet since API calls are blocked.
	assert.NotContains(out.String(), "[open]")

	// Release the API calls.
	close(apiGate)

	testutil.WaitForOutput(t, out, "[open]")

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
		"check,COMPLETED,SUCCESS,\nstate,OPEN\nreviewRequestCount,1\nlatestReview,someuser,APPROVED,4,0\nmergeStateStatus,CLEAN\nisDraft,false\nautoMerge,false",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

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
		"headCommit,deadbeef\ncheck,COMPLETED,SUCCESS,\nstate,OPEN\nreviewRequestCount,1\nlatestReview,alice,CHANGES_REQUESTED,0,0,deadbeef\nlatestReview,bob,APPROVED,50,0,deadbeef\nmergeStateStatus,BLOCKED\nisDraft,false\nautoMerge,false",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.Contains(out, "[approved: 0/3]")
	assert.Contains(out, "alice requested changes")
	assert.Contains(out, "bob approved with comments")
	assert.NotContains(out, "[can merge]")
}

func TestSdLog_WhenStatusFlag_IgnoresCommentsOnOlderCommit(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	// bob's approving comments are on an older commit (staleoid), so they are
	// stale relative to the PR head (deadbeef) and should not surface.
	testExecutor.SetResponse(
		"headCommit,deadbeef\ncheck,COMPLETED,SUCCESS,\nstate,OPEN\nreviewRequestCount,0\nlatestReview,bob,APPROVED,50,0,staleoid\nmergeStateStatus,CLEAN\nisDraft,false\nautoMerge,false",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.Contains(out, "bob approved")
	assert.NotContains(out, "approved with comments")
	assert.Contains(out, "[approved: 1/1]")
}

func TestSdLog_WhenStatusFlag_IgnoresCommentedReviewOnOlderCommit(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	// richa's COMMENTED review has no commit oid (empty), so it is not on the PR
	// head (deadbeef) and should not surface as "commented".
	testExecutor.SetResponse(
		"headCommit,deadbeef\ncheck,COMPLETED,SUCCESS,\nstate,OPEN\nreviewRequestCount,0\nlatestReview,richa,COMMENTED,50,0,\nmergeStateStatus,CLEAN\nisDraft,false\nautoMerge,false",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.NotContains(out, "richa commented")
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
		"check,COMPLETED,SUCCESS,\nstate,OPEN\nreviewRequestCount,0\nlatestReview,alice,CHANGES_REQUESTED,0,0\nlatestReview,bob,CHANGES_REQUESTED,0,0\nlatestReview,carol,APPROVED,0,0\nlatestReview,dave,APPROVED,0,0\nmergeStateStatus,BLOCKED\nisDraft,false\nautoMerge,false",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

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
		"check,COMPLETED,SUCCESS,\nstate,MERGED\nreviewRequestCount,0\nlatestReview,someuser,APPROVED,4,0\nmergeStateStatus,CLEAN\nisDraft,false\nautoMerge,false",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.Contains(out, "[merged]")
	assert.Contains(out, "[checks: passed")
}

func TestSdLog_WhenStatusFlag_ShowsMergingStatus(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse(
		"check,COMPLETED,SUCCESS,\nstate,OPEN\nreviewRequestCount,0\nlatestReview,someuser,APPROVED,4,0\nmergeStateStatus,CLEAN\nisDraft,false\nautoMerge,true",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.Contains(out, "[merging]")
	assert.NotContains(out, "[open]")
	assert.NotContains(out, "[merged]")
}

func TestSdLog_WhenStatusFlag_ShowsMergeQueueStatus(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	gitutil.ResetCacheForTesting()
	testExecutor.SetResponse(
		"myowner/myrepo",
		nil, "gh", "repo", "view", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse(
		"check,COMPLETED,SUCCESS,\nstate,OPEN\nnumber,42\nreviewRequestCount,0\nlatestReview,someuser,APPROVED,4,0\nmergeStateStatus,CLEAN\nisDraft,false\nautoMerge,true",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.Contains(out, "[merging]")
	assert.NotContains(out, "[open]")
}

func TestSdLog_WhenStatusFlagAndNoPRs_PrintsCommitsWithoutBubbletea(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")

	out := testParseArguments("log", "--status")

	assert.Contains(out, "first")
	assert.Contains(out, "second")
	assert.NotContains(out, "✓")
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
		"state,OPEN\nreviewRequestCount,0\nmergeStateStatus,BLOCKED\nisDraft,true\nautoMerge,false",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

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

	// Create a secondary worktree (automatically creates branch "secondary-worktree").
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../secondary-worktree")

	if err := os.Chdir("../secondary-worktree"); err != nil {
		t.Fatal(err)
	}
	testutil.AddCommit("secondary-commit", "")

	// Go back to main worktree.
	if err := os.Chdir(mainPath); err != nil {
		t.Fatal(err)
	}
	gitutil.ResetCacheForTesting()

	out := testParseArguments("log", "-c", "showWorktrees=true")

	assert.Contains(out, "main-commit")
	assert.Contains(out, "secondary-commit")
	assert.Contains(out, "secondary-worktree")
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

	// Create a secondary worktree (automatically creates branch "secondary-worktree").
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../secondary-worktree")

	if err := os.Chdir("../secondary-worktree"); err != nil {
		t.Fatal(err)
	}
	testutil.AddCommit("secondary-commit", "")

	// Go back to main worktree.
	if err := os.Chdir(mainPath); err != nil {
		t.Fatal(err)
	}
	gitutil.ResetCacheForTesting()

	testExecutor.SetResponse(
		"abc123def456abc123def456abc123def456abc123",
		nil, "git", "log", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse(
		"check,COMPLETED,SUCCESS,\nstate,OPEN\nreviewRequestCount,0\nmergeStateStatus,CLEAN\nisDraft,false\nautoMerge,false",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "-c", "showWorktrees=true", "--status")

	assert.Contains(out, "main-commit")
	assert.Contains(out, "secondary-commit")
	assert.Contains(out, "secondary-worktree")
}

func TestSdLog_WhenOtherWorktreeCommitHasBranch_ShowsCheckAndBranchCommits(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("main-commit", "")

	mainPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	// Create a secondary worktree (automatically creates branch "secondary-worktree").
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../secondary-worktree")

	if err := os.Chdir("../secondary-worktree"); err != nil {
		t.Fatal(err)
	}
	testutil.AddCommit("secondary-commit", "")

	// Create a PR for the worktree commit.
	testParseArguments("new", "1")

	testutil.AddCommit("extra-on-branch", "")
	testParseArguments("update", "2", "1")

	// Go back to main worktree.
	if err := os.Chdir(mainPath); err != nil {
		t.Fatal(err)
	}
	gitutil.ResetCacheForTesting()

	out := testParseArguments("log", "-c", "showWorktrees=true")

	assert.Contains(out, "main-commit")
	assert.Contains(out, "secondary-commit")
	assert.Contains(out, "✓")
	assert.Contains(out, "extra-on-branch")
	assert.Contains(out, "secondary-worktree")
}

func TestSdLog_WhenOtherWorktreeHasOnlySharedCommits_DoesNotShowWorktreeSection(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("shared-commit", "")

	// Create a secondary worktree that has the same commits as main (no unique ones).
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../secondary-worktree")

	gitutil.ResetCacheForTesting()

	out := testParseArguments("log")

	assert.Contains(out, "shared-commit")
	assert.NotContains(out, "secondary-worktree")
}

func writeGhStackFile(t *testing.T, branches ...string) {
	var entries []string
	for _, branch := range branches {
		entries = append(entries, `{"branch":"`+branch+`"}`)
	}
	contents := `{"schemaVersion":1,"stacks":[{"branches":[` + strings.Join(entries, ",") + `]}]}`
	gitDir := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "--absolute-git-dir")
	assert.NoError(t, os.WriteFile(filepath.Join(gitDir, "gh-stack"), []byte(contents), 0o644))
}

func TestSdLog_WhenBranchInGhStack_PrintsStackEmojiAndAlignsOtherRows(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testParseArguments("new", "2")
	testParseArguments("new", "1")
	firstBranch := strings.TrimSpace(testParseArguments("branch-name", "2"))
	writeGhStackFile(t, firstBranch, "other")

	out := testParseArguments("log")

	assert.Regexp("🥞 .*first", out)
	assert.NotRegexp("✓.*first", out)
	assert.NotRegexp("🥞.*second", out)
	assert.Contains(out, "✓  ")
}

func TestSdLog_WhenNoGhStack_DoesNotPrintStackEmoji(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")

	out := testParseArguments("log")

	assert.NotContains(out, "🥞")
}

func TestSdLog_WhenBranchDirectlyAboveAnotherInGhStack_PrintsArrowWithoutIndex(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testParseArguments("new", "2")
	testParseArguments("new", "1")
	firstBranch := strings.TrimSpace(testParseArguments("branch-name", "2"))
	secondBranch := strings.TrimSpace(testParseArguments("branch-name", "1"))
	writeGhStackFile(t, firstBranch, secondBranch)

	out := testParseArguments("log")

	assert.Regexp("(?m)^\\s+⬇️$", out)
	assert.Equal(1, strings.Count(out, "⬇️"))
}

func TestSdLog_WhenBranchNotDirectlyAboveAnotherInGhStack_PrintsArrowWithIndex(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("middle", "")
	testutil.AddCommit("third", "")
	testParseArguments("new", "3")
	testParseArguments("new", "1")
	firstBranch := strings.TrimSpace(testParseArguments("branch-name", "3"))
	thirdBranch := strings.TrimSpace(testParseArguments("branch-name", "1"))
	writeGhStackFile(t, firstBranch, thirdBranch)

	out := testParseArguments("log")

	assert.Contains(out, "⬇️ 3")
}

func TestSdLog_WhenPreviousGhStackBranchNotInLog_PrintsQuestionMark(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")
	firstBranch := strings.TrimSpace(testParseArguments("branch-name", "1"))
	writeGhStackFile(t, "deleted-branch", firstBranch)

	out := testParseArguments("log")

	assert.Contains(out, "⬇️ ❓")
}

func TestSdLog_WhenBranchAtBottomOfGhStack_DoesNotPrintArrow(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")
	firstBranch := strings.TrimSpace(testParseArguments("branch-name", "1"))
	writeGhStackFile(t, firstBranch)

	out := testParseArguments("log")

	assert.NotContains(out, "⬇️")
}

func TestSdLog_WhenStatusFlagAndBranchesInGhStack_ShowsStackEmojiAndArrow(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testParseArguments("new", "2")
	testParseArguments("new", "1")
	firstBranch := strings.TrimSpace(testParseArguments("branch-name", "2"))
	secondBranch := strings.TrimSpace(testParseArguments("branch-name", "1"))
	writeGhStackFile(t, firstBranch, secondBranch)
	testExecutor.SetResponse(
		"state,OPEN\nreviewRequestCount,0\nmergeStateStatus,BLOCKED\nisDraft,false\nautoMerge,false",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	out := testParseArguments("log", "--status")

	assert.Regexp("🥞 .*first", out)
	assert.Regexp("🥞 .*second", out)
	assert.Regexp("(?m)^\\s+⬇️\\s*$", out)
	assert.Equal(1, strings.Count(out, "⬇️"))
}

func TestSdLog_WhenOnBranchAboveAnotherInGhStack_OnlyShowsCommitsSincePreviousBranch(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "stack-1")
	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "stack-2")
	testutil.AddCommit("second", "")
	writeGhStackFile(t, "stack-1", "stack-2", "stack-3")

	out := testParseArguments("log")

	assert.NotContains(out, "first")
	assert.Contains(out, "second")
	assert.Contains(out, "⬇️ stack-1")
	assert.Less(strings.Index(out, "⬆️ stack-3"), strings.Index(out, "🥞 (2/3)"))
	assert.Less(strings.Index(out, "🥞 (2/3)"), strings.Index(out, "second"))
}

func TestSdLog_WhenOnBottomBranchOfGhStack_ShowsCommitsSinceMain(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "stack-1")
	testutil.AddCommit("first", "")
	writeGhStackFile(t, "stack-1", "stack-2")

	out := testParseArguments("log")

	assert.Contains(out, "first")
	assert.NotContains(out, "⬇️")
	assert.Contains(out, "⬆️ stack-2")
	assert.Contains(out, "🥞 (1/2)")
}

func TestSdLog_WhenMainFlagOnFeatureBranch_PrintsMainLogs(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature")
	testutil.AddCommit("second", "")

	out := testParseArguments("log", "--main")

	assert.Contains(out, "first")
	assert.NotContains(out, "second")
}
