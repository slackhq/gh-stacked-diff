package commands

import (
	"log/slog"
	"strings"
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
	testExecutor.SetResponse("rateLimit,1,4999,5000,2025-01-01T00:00:00Z\nisDraft,true\nstate,OPEN\nnumber,1\nreviewRequestCount,0\nmergeStateStatus,BLOCKED",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	// Select the commit in the update dialog (enter selects current row and confirms)
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))

	testParseArguments("update-branches")

	// Verify branch was updated with the amended commit
	branchFileContent := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "show", allCommits[0].Branch+":file1")
	assert.Equal("amended", branchFileContent)
}

func TestSdUpdateBranches_NonDraftBranch_MergesOriginBranch(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Add "first" commit and create PR branch
	testutil.CommitFileChange("first", "file1", "original")
	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()

	// Push a new commit to a different file on origin/branch (simulates review fixup pushed remotely)
	gitutil.GitSwitch(allCommits[0].Branch)
	testutil.CommitFileChange("fixup", "file2", "review-fix")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", allCommits[0].Branch)
	// Reset local branch back so it diverges from origin
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", "HEAD~1")
	gitutil.GitSwitch(gitutil.GetLocalMainBranchOrDie())

	// Amend the commit on main so its diff differs from the branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--soft", "HEAD~1")
	testutil.CommitFileChange("first", "file1", "amended")

	// Mock PR status as NOT draft
	testExecutor.SetResponse("rateLimit,1,4999,5000,2025-01-01T00:00:00Z\nisDraft,false\nstate,OPEN\nnumber,1\nreviewRequestCount,0\nmergeStateStatus,CLEAN",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	// Select the commit in the update dialog
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))

	testParseArguments("update-branches")

	// Verify origin/branch changes were merged in (file2 from the remote fixup)
	branchFile2Content := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "show", allCommits[0].Branch+":file2")
	assert.Equal("review-fix", branchFile2Content)

	// Verify the commit diff was applied (file1 has the amended content)
	branchFile1Content := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "show", allCommits[0].Branch+":file1")
	assert.Equal("amended", branchFile1Content)
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

func TestSdUpdateBranches_NoCommitsAhead_LogsAndReturns(t *testing.T) {
	_ = testutil.InitTest(t, slog.LevelError)

	// No commits ahead of origin/main, so nothing to do
	out := testParseArguments("update-branches")
	assert.Contains(t, out, "No commits ahead of origin/")
}

func TestSdUpdateBranches_NonDraftMergeConflict_AppliesCommitDiff(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelDebug)

	// Set up base file in origin/main
	testutil.CommitFileChange("add-file1", "file1", "base")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Add an intermediate commit (so cherry-pick produces a different hash)
	testutil.AddCommit("commit-a", "file2")
	// Create the commit that modifies file1 and create PR branch
	testutil.CommitFileChange("commit-b", "file1", "original")
	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()

	// Simulate a reviewer pushing a fixup to the PR branch on origin
	gitutil.GitSwitch(allCommits[0].Branch)
	testutil.CommitFileChange("reviewer-fixup", "file1", "review-change")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", allCommits[0].Branch)
	gitutil.GitSwitch(gitutil.GetLocalMainBranchOrDie())

	// Advance origin/main with a conflicting change to file1
	testutil.CommitFileChange("origin-advance", "file1", "origin-content")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Amend the commit on main so its diff differs from the branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--soft", "HEAD~1")
	testutil.CommitFileChange("commit-b", "file1", "my-amended-content")

	allCommits = templates.GetAllCommits()

	// Mock PR status as NOT draft
	testExecutor.SetResponse("rateLimit,1,4999,5000,2025-01-01T00:00:00Z\nisDraft,false\nstate,OPEN\nnumber,1\nreviewRequestCount,0\nmergeStateStatus,CLEAN",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))

	testParseArguments("update-branches")

	// Merge conflict in file1 is resolved (file1 is in commit diff), then cherry-pick applies amended content
	branchFileContent := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "show", allCommits[0].Branch+":file1")
	assert.Equal("my-amended-content", branchFileContent)

	// Verify it's a merge commit (merge succeeded, not rebase fallback)
	branchLog := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "log", "--format=%s", allCommits[0].Branch)
	assert.Contains(branchLog, "Merge")
}

func TestSdUpdateBranches_NonDraftMergeConflictInUnrelatedFile_FallsBackToRebase(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Set up base files in origin/main
	testutil.CommitFileChange("add-file1", "file1", "base")
	testutil.CommitFileChange("add-file2", "file2", "base2")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Add an intermediate commit that modifies file2 (so mergeBase includes file2 changes)
	testutil.CommitFileChange("intermediate", "file2", "intermediate-content")
	// Create a commit that only modifies file1 (not file2)
	testutil.CommitFileChange("commit-b", "file1", "my-content")
	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()

	// Simulate a reviewer pushing a conflicting change to file2 on the PR branch.
	// The branch was created from mergeBase (add-file2) so it has file2="base2".
	// The reviewer changes it to something that will conflict with "intermediate-content".
	gitutil.GitSwitch(allCommits[0].Branch)
	testutil.CommitFileChange("reviewer-fixup", "file2", "branch-file2-content")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", allCommits[0].Branch)
	gitutil.GitSwitch(gitutil.GetLocalMainBranchOrDie())

	// Advance origin/main so mergeBase stays at "commit-b" level
	testutil.CommitFileChange("origin-advance", "file3", "other")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Amend the commit on main so the diff differs from the branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--soft", "HEAD~1")
	testutil.CommitFileChange("commit-b", "file1", "my-amended-content")

	allCommits = templates.GetAllCommits()

	// Mock PR status as NOT draft
	testExecutor.SetResponse("rateLimit,1,4999,5000,2025-01-01T00:00:00Z\nisDraft,false\nstate,OPEN\nnumber,1\nreviewRequestCount,0\nmergeStateStatus,CLEAN",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))

	testParseArguments("update-branches")

	// Should have fallen back to rebase - branch has amended content but no merge commit
	branchFileContent := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "show", allCommits[0].Branch+":file1")
	assert.Equal("my-amended-content", branchFileContent)

	// Verify it's NOT a merge commit (rebase creates a linear history)
	branchLog := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "log", "--format=%s", allCommits[0].Branch)
	assert.NotContains(branchLog, "Merge")
}

func TestSdUpdateBranches_CherryPickFails_SkipsBranch(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create an initial file so we can cause a conflict
	testutil.CommitFileChange("setup", "file1", "base-content")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create a commit that modifies file1
	testutil.CommitFileChange("first", "file1", "commit-content")
	testParseArguments("new", "1")

	// Push a conflicting change to origin/main so cherry-pick onto mergeBase will conflict
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", "HEAD~1")
	testutil.CommitFileChange("conflicting", "file1", "conflicting-content")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Recreate our commit on top
	testutil.CommitFileChange("first", "file1", "commit-content-v2")

	// Mock PR status as draft
	testExecutor.SetResponse("rateLimit,1,4999,5000,2025-01-01T00:00:00Z\nisDraft,true\nstate,OPEN\nnumber,1\nreviewRequestCount,0\nmergeStateStatus,BLOCKED",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))

	testParseArguments("update-branches")

	// Verify we're back on main (command didn't panic)
	currentBranch := strings.TrimSpace(util.ExecuteOrDie(util.ExecuteOptions{}, "git", "rev-parse", "--abbrev-ref", "HEAD"))
	assert.Equal(gitutil.GetLocalMainBranchOrDie(), currentBranch)
}

func TestSdUpdateBranches_CommitWithoutBranch_IsSkipped(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create a commit WITH a PR branch
	testutil.CommitFileChange("with-branch", "file1", "original")
	testParseArguments("new", "1")

	// Create a second commit WITHOUT a PR branch (no `sd new`)
	testutil.CommitFileChange("no-branch", "file2", "content")

	allCommits := templates.GetAllCommits()
	// allCommits[0] = "no-branch" (newest, no PR), allCommits[1] = "with-branch" (has PR)
	branchedCommit := allCommits[1]

	// Amend the branched commit so its diff differs from the branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--soft", "HEAD~2")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "file2")
	testutil.CommitFileChange("with-branch", "file1", "amended")
	testutil.CommitFileChange("no-branch", "file2", "content-changed")

	// Mock PR status as draft for the branched commit
	testExecutor.SetResponse("rateLimit,1,4999,5000,2025-01-01T00:00:00Z\nisDraft,true\nstate,OPEN\nnumber,1\nreviewRequestCount,0\nmergeStateStatus,BLOCKED",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	// Only the branched commit is selectable, so enter selects it and confirms
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))

	testParseArguments("update-branches")

	// Verify the branched commit was updated with the amended content
	branchFileContent := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "show", branchedCommit.Branch+":file1")
	assert.Equal("amended", branchFileContent)

	// Verify no branch exists for the branchless commit
	branches := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch")
	assert.NotContains(branches, "no-branch")
}

func TestSdUpdateBranches_BranchDiffMatchesButBehindMain_IsSelectable(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	mainBranch := gitutil.GetLocalMainBranchOrDie()

	// Step 1: Create the "first" commit and its PR branch.
	// origin/main is still at the initial commit (A), so the branch is based at A.
	//   origin/main: A
	//   local main:  A -> B ("first")
	//   branch:      A -> D (cherry-pick of B)
	testutil.CommitFileChange("first", "file1", "original")
	testParseArguments("new", "1")
	branchName := templates.GetAllCommits()[0].Branch

	// Step 2: Advance origin/main to Z via a side path that does NOT include B.
	// This simulates a colleague pushing an unrelated commit to origin/main.
	//   origin/main after: A -> Z ("unrelated", no file1)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "tmp-advance", "HEAD~1")
	testutil.CommitFileChange("unrelated", "file2", "unrelated-content")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", "tmp-advance:"+mainBranch)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", mainBranch)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "-D", "tmp-advance")

	// Step 3: Fetch and rebase local main onto origin/main (simulates `sd rebase-main`).
	// B (adds file1) replays on top of Z, producing E.
	//   origin/main: A -> Z
	//   local main:  A -> Z -> E (rebased "first", adds file1 with "original")
	//   branch:      A -> D (still based at A, not updated yet)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "fetch", "origin")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "rebase", "origin/"+mainBranch)

	// Verify preconditions (state is as expected):
	//   commitDiff(E) = adds file1 with "original"
	//   branchDiff(D from merge-base=A) = adds file1 with "original"
	//   → branchNeedsUpdate condition 1 = false (diffs match)
	//   mainMergeBase = merge-base(origin/main=Z, local-main=E) = Z
	//   IsAncestor(Z, D): Z is NOT in D's ancestry (D's parent is A)
	//   → branchNeedsUpdate condition 2 = true → branch is selectable

	// Mock PR status as draft
	testExecutor.SetResponse("rateLimit,1,4999,5000,2025-01-01T00:00:00Z\nisDraft,true\nstate,OPEN\nnumber,1\nreviewRequestCount,0\nmergeStateStatus,BLOCKED",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	// Select the commit in the update dialog
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))

	testParseArguments("update-branches")

	// Verify branch was updated: parent should now be origin/main tip (Z = merge-base).
	// updateWithRebase does: git branch -f branch Z, then cherry-pick E onto Z.
	originMainTip := strings.TrimSpace(util.ExecuteOrDie(util.ExecuteOptions{}, "git", "rev-parse", "origin/"+mainBranch))
	branchParent := strings.TrimSpace(util.ExecuteOrDie(util.ExecuteOptions{}, "git", "rev-parse", branchName+"^1"))
	assert.Equal(originMainTip, branchParent, "branch parent should be origin/main tip (Z) after update")

	// And file1 still has the original content (diff was preserved via cherry-pick)
	branchFileContent := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "show", branchName+":file1")
	assert.Equal("original", branchFileContent)
}

func TestSdUpdateBranches_MultipleBranches_ContinuesAfterFailure(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create two commits and branches
	testutil.AddCommit("first", "file1")
	testParseArguments("new", "1")
	testutil.AddCommit("second", "file2")
	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	// allCommits[0] = "second" (newest), allCommits[1] = "first" (oldest)

	// Push current main to origin so branches diverge from origin/main
	// (branches were created from the old mergeBase, now origin/main is ahead)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Mock both PRs as draft
	testExecutor.SetResponse("rateLimit,1,4999,5000,2025-01-01T00:00:00Z\nisDraft,true\nstate,OPEN\nnumber,2\nreviewRequestCount,0\nmergeStateStatus,BLOCKED",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse("rateLimit,1,4999,5000,2025-01-01T00:00:00Z\nisDraft,true\nstate,OPEN\nnumber,1\nreviewRequestCount,0\nmergeStateStatus,BLOCKED",
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	// Delete the "first" branch from origin so push fails (causing updateWithRebase to panic)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", "--delete", allCommits[1].Branch)

	// Select all in the dialog
	interactive.SendToProgram(0,
		interactive.NewMessageKey(tea.KeySpace),
		interactive.NewMessageKey(tea.KeyDown),
		interactive.NewMessageKey(tea.KeySpace),
		interactive.NewMessageKey(tea.KeyEnter),
	)

	testParseArguments("update-branches")

	// "second" branch (allCommits[0]) should still be updated despite "first" failing.
	// Verify that branch tip commit now matches the local commit.
	branchTip := strings.TrimSpace(util.ExecuteOrDie(util.ExecuteOptions{}, "git", "rev-parse", allCommits[0].Branch))
	localCommit := allCommits[0].Commit
	assert.NotEqual(localCommit, branchTip, "branch should have been rebased to a new commit")

	// Verify we're back on main
	currentBranch := strings.TrimSpace(util.ExecuteOrDie(util.ExecuteOptions{}, "git", "rev-parse", "--abbrev-ref", "HEAD"))
	assert.Equal(gitutil.GetLocalMainBranchOrDie(), currentBranch)
}
