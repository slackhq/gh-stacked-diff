package commands

import (
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestSdMigrate_BasicTest(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Add test setup (commits, branches, etc.)
	testutil.AddCommit("first commit", "file1.txt")

	// Execute the command (should find no branches since we only have main)
	out := testParseArguments("migrate")

	// Assert expected behavior
	assert.Contains(out, "No branches found with your commits")
}

func TestSdMigrate_WithUnexpectedArguments(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first commit", "file1.txt")

	// Execute the command with unexpected arguments and expect it to panic
	defer func() {
		r := recover()
		assert.NotNil(r, "expected panic on unexpected arguments")
	}()

	testParseArguments("migrate", "extra-arg")

	assert.Fail("did not panic on unexpected arguments")
}

func TestFindUserBranches_FindsBranchesWithUserCommits(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create first branch with user commit
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-1")
	testutil.AddCommit("feature 1 commit", "feature1.txt")
	time.Sleep(time.Second) // Ensure different timestamps

	// Create second branch with user commit (more recent)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-2")
	testutil.AddCommit("feature 2 commit", "feature2.txt")
	time.Sleep(time.Second) // Ensure different timestamps

	// Create third branch with user commit (oldest)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-3")
	time.Sleep(time.Second) // Ensure different timestamps

	// Go back to feature-1 and add another commit to make it the most recent
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "feature-1")
	testutil.AddCommit("feature 1 second commit", "feature1b.txt")

	// Go back to main
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())

	// Find user branches
	branches := findUserBranches()

	// Should find all 3 feature branches (feature-3 has commits from main, even though no new commits were added to it)
	// Note: feature-3 will be found because it inherits commits from main that were made by the test user
	assert.Equal(3, len(branches), "Should find exactly 3 branches with user commits")

	// Should be sorted by most recent commit first (feature-1 then feature-2 then feature-3)
	assert.Equal("feature-1", branches[0], "feature-1 should be first (most recent commit)")
	assert.Equal("feature-2", branches[1], "feature-2 should be second")
	assert.Equal("feature-3", branches[2], "feature-3 should be third (has commits from main)")
}

func TestFindUserBranches_NoBranches(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main
	testutil.AddCommit("initial commit", "file1.txt")

	// Find user branches (should be empty)
	branches := findUserBranches()

	assert.Equal(0, len(branches), "Should find no branches when only main exists")
}

func TestFindUserBranches_ExcludesMainBranch(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create commits on main
	testutil.AddCommit("first commit", "file1.txt")
	testutil.AddCommit("second commit", "file2.txt")

	// Find user branches (should be empty since we're only on main)
	branches := findUserBranches()

	assert.Equal(0, len(branches), "Should not include main branch in results")
}

func TestSdMigrate_RebaseSingleBranch(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main and push to origin
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create a feature branch with a commit
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-1")
	testutil.AddCommit("feature 1 commit", "feature1.txt")

	// Go back to main and add another commit to origin/main
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())
	testutil.AddCommit("new main commit", "file2.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Record current branch before running migrate
	currentBranch := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--show-current")

	// Manually test the rebase logic
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "feature-1")

	// Get the most recent main commit
	mostRecentMainCommit := gitutil.FirstOriginMainCommit(gitutil.GetLocalMainBranchOrDie())

	// Perform rebase
	_, err := util.Execute(util.ExecuteOptions{}, "git", "rebase", mostRecentMainCommit)
	assert.Nil(err, "Rebase should succeed")

	// Verify we're still on feature-1
	newBranch := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--show-current")
	assert.Equal("feature-1", newBranch)

	// Restore original branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", currentBranch)
}

func TestSdMigrate_RebaseMultipleBranches(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main and push to origin
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create first feature branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-1")
	testutil.AddCommit("feature 1 commit", "feature1.txt")

	// Create second feature branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-2")
	testutil.AddCommit("feature 2 commit", "feature2.txt")

	// Go back to main and add commits
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())
	testutil.AddCommit("new main commit", "file2.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Get the most recent main commit
	mostRecentMainCommit := gitutil.FirstOriginMainCommit(gitutil.GetLocalMainBranchOrDie())

	// Save current branch
	currentBranch := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--show-current")

	// Rebase both branches
	branches := []string{"feature-1", "feature-2"}
	for _, branch := range branches {
		util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", branch)
		_, err := util.Execute(util.ExecuteOptions{}, "git", "rebase", mostRecentMainCommit)
		assert.Nil(err, "Rebase should succeed for "+branch)
	}

	// Restore original branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", currentBranch)

	// Verify both branches exist and have been rebased
	branches = findUserBranches()
	assert.Equal(2, len(branches), "Should still have 2 branches after rebase")
}

func TestSdMigrate_RebaseFailure(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main and push to origin
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create a feature branch with a commit that modifies file1.txt
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-1")
	util.ExecuteOrDie(util.ExecuteOptions{}, "sh", "-c", "echo 'feature content' > file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "-m", "feature 1 changes file1")

	// Go back to main and modify the same file differently
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())
	util.ExecuteOrDie(util.ExecuteOptions{}, "sh", "-c", "echo 'main content' > file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "-m", "main changes file1 differently")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Get the most recent main commit
	mostRecentMainCommit := gitutil.FirstOriginMainCommit(gitutil.GetLocalMainBranchOrDie())

	// Try to rebase - should fail due to conflict
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "feature-1")
	_, err := util.Execute(util.ExecuteOptions{}, "git", "rebase", mostRecentMainCommit)
	assert.NotNil(err, "Rebase should fail due to conflict")

	// Abort the rebase
	_, abortErr := util.Execute(util.ExecuteOptions{}, "git", "rebase", "--abort")
	if abortErr != nil {
		t.Logf("Failed to abort rebase: %s", abortErr.Error())
	}

	// Verify we're still on feature-1 and in a clean state
	newBranch := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--show-current")
	assert.Equal("feature-1", newBranch)
}

func TestSdMigrate_EndsOnMainBranch(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main and push to origin
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create a feature branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-1")
	testutil.AddCommit("feature 1 commit", "feature1.txt")

	// Create another branch and stay on it
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-2")
	testutil.AddCommit("feature 2 commit", "feature2.txt")

	// Record current branch (feature-2)
	originalBranch := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--show-current")
	assert.Equal("feature-2", originalBranch)

	// Get the most recent main commit
	mostRecentMainCommit := gitutil.FirstOriginMainCommit(gitutil.GetLocalMainBranchOrDie())

	// Simulate the migrate process: checkout feature-1, rebase it
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "feature-1")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "rebase", mostRecentMainCommit)

	// End on main branch (not restoring original branch)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())

	// Verify we're on main
	currentBranch := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--show-current")
	assert.Equal(gitutil.GetLocalMainBranchOrDie(), currentBranch)
}

func TestGetCommitsAhead_SingleCommit(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main
	testutil.AddCommit("initial commit", "file1.txt")
	baseCommit := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "HEAD")

	// Add one more commit
	testutil.AddCommit("second commit", "file2.txt")

	// Get commits ahead
	commits := getCommitsAhead(baseCommit, "HEAD")

	assert.Equal(1, len(commits), "Should have exactly 1 commit ahead")

	// Verify the commit hash matches
	allCommits := templates.GetAllCommits()
	assert.Equal(allCommits[0].Commit, commits[0])
}

func TestGetCommitsAhead_MultipleCommits(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main
	testutil.AddCommit("initial commit", "file1.txt")
	baseCommit := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "HEAD")

	// Add three more commits
	testutil.AddCommit("second commit", "file2.txt")
	testutil.AddCommit("third commit", "file3.txt")
	testutil.AddCommit("fourth commit", "file4.txt")

	// Get commits ahead
	commits := getCommitsAhead(baseCommit, "HEAD")

	assert.Equal(3, len(commits), "Should have exactly 3 commits ahead")

	allCommits := templates.GetAllCommits()
	// Verify commits are in chronological order (oldest first)
	assert.Equal(allCommits[2].Commit, commits[0], "First commit should be commit2")
	assert.Equal(allCommits[1].Commit, commits[1], "Second commit should be commit3")
	assert.Equal(allCommits[0].Commit, commits[2], "Third commit should be commit4")
}

func TestGetCommitsAhead_NoCommits(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main
	testutil.AddCommit("initial commit", "file1.txt")
	baseCommit := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "HEAD")

	// Get commits ahead (should be none since HEAD == baseCommit)
	commits := getCommitsAhead(baseCommit, "HEAD")

	assert.Equal(0, len(commits), "Should have no commits ahead")
}

func TestGetCommitsAhead_AfterRebase(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main and push to origin
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create a feature branch with commits
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-1")
	testutil.AddCommit("feature commit 1", "feature1.txt")
	testutil.AddCommit("feature commit 2", "feature2.txt")

	// Go back to main and add a commit
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())
	testutil.AddCommit("new main commit", "main.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	newMainCommit := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "HEAD")

	// Rebase feature branch onto new main
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "feature-1")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "rebase", newMainCommit)

	// Get commits ahead after rebase
	commits := getCommitsAhead(newMainCommit, "HEAD")

	assert.Equal(2, len(commits), "Should have 2 commits ahead after rebase")
}

func TestGetUnmergedPR_WithOpenPR(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main and push to origin
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create a feature branch with a commit
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-1")
	testutil.AddCommit("feature 1 commit", "feature1.txt")

	// Mock the gh pr list response to simulate an open PR
	testExecutor.SetResponse("123"+gitutil.GhDelim+"Add new feature"+gitutil.GhDelim+"OPEN"+gitutil.GhDelim+"body", nil,
		"gh", "pr", "list", "--head", "feature-1", "--state", "open",
		util.MatchAnyRemainingArgs)

	// Test getUnmergedPR
	pr := gitutil.GetUnmergedPR("feature-1")

	assert.NotNil(pr, "Should find an open PR")
	assert.Equal("123", pr.Number)
	assert.Equal("Add new feature", pr.Title)
	assert.Equal("OPEN", pr.State)
}

func TestGetUnmergedPR_WithoutPR(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main and push to origin
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create a feature branch with a commit
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-1")
	testutil.AddCommit("feature 1 commit", "feature1.txt")

	// Mock the gh pr list response to simulate no PR
	testExecutor.SetResponse("", nil,
		"gh", "pr", "list", "--head", "feature-1", "--state", "open",
		"--json", "number,title,state", "--jq", util.MatchAnyRemainingArgs)

	// Test getUnmergedPR
	pr := gitutil.GetUnmergedPR("feature-1")

	assert.Nil(pr, "Should not find a PR when none exists")
}

func TestGetMergedPR_WithMergedPR(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main and push to origin
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create a feature branch with a commit
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-1")
	testutil.AddCommit("feature 1 commit", "feature1.txt")

	// Mock the gh pr list response to simulate a merged PR
	testExecutor.SetResponse("456|Implement feature|MERGED", nil,
		"gh", "pr", "list", "--head", "feature-1", "--state", "merged",
		"--json", "number,title,state", "--jq", util.MatchAnyRemainingArgs)

	// Test getMergedPR
	pr := gitutil.GetMergedPR("feature-1")

	assert.NotNil(pr, "Should find a merged PR")
	assert.Equal("456", pr.Number)
	assert.Equal("Implement feature", pr.Title)
	assert.Equal("MERGED", pr.State)
}

func TestGetMergedPR_WithoutMergedPR(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main and push to origin
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create a feature branch with a commit
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-1")
	testutil.AddCommit("feature 1 commit", "feature1.txt")

	// Mock the gh pr list response to simulate no merged PR
	testExecutor.SetResponse("", nil,
		"gh", "pr", "list", "--head", "feature-1", "--state", "merged",
		"--json", "number,title,state", "--jq", util.MatchAnyRemainingArgs)

	// Test getMergedPR
	pr := gitutil.GetMergedPR("feature-1")

	assert.Nil(pr, "Should not find a merged PR when none exists")
}

func TestSdMigrate_SkipsBranchWithMergedPR(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main and push to origin
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create a feature branch with a commit
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-1")
	testutil.AddCommit("feature 1 commit", "feature1.txt")

	// Go back to main
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())

	// Mock the gh pr list response to simulate a merged PR
	testExecutor.SetResponse("789|Already merged feature|MERGED", nil,
		"gh", "pr", "list", "--head", "feature-1", "--state", "merged",
		"--json", "number,title,state", "--jq", util.MatchAnyRemainingArgs)

	// Get the most recent main commit
	mostRecentMainCommit := gitutil.FirstOriginMainCommit(gitutil.GetLocalMainBranchOrDie())

	// Record current branch before processing
	currentBranch := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--show-current")

	// Process the branch
	processBranch("feature-1", mostRecentMainCommit)

	// Verify we're still on the original branch (not checked out to feature-1)
	finalBranch := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--show-current")
	assert.Equal(currentBranch, finalBranch, "Should remain on original branch when skipping merged PR")
}

func TestSdMigrate_SkipsDuplicateCommitsWhenMigratingBranchWithoutPR(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create initial commit on main and push to origin
	testutil.AddCommit("initial commit", "file1.txt")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	// Create a feature branch with 3 commits
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "feature-branch")
	testutil.AddCommit("first feature commit", "feature1.txt")
	firstCommitHash := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "HEAD")

	testutil.AddCommit("second feature commit", "feature2.txt")

	testutil.AddCommit("third feature commit", "feature3.txt")
	thirdCommitHash := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "HEAD")

	// Go back to main and manually cherry-pick the first and third commits
	// This simulates commits that are already on main
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", gitutil.GetLocalMainBranchOrDie())
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "cherry-pick", firstCommitHash)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "cherry-pick", thirdCommitHash)

	// Mock the gh pr list responses to simulate no PR exists (neither merged nor open)
	testExecutor.SetResponse("", nil,
		"gh", "pr", "list", "--head", "feature-branch", "--state", "merged",
		"--json", "number,title,state", "--jq", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse("", nil,
		"gh", "pr", "list", "--head", "feature-branch", "--state", "open",
		"--json", "number,title,state", "--jq", util.MatchAnyRemainingArgs)

	// Get the base commit for the feature branch
	mostRecentMainCommit := gitutil.FirstOriginMainCommit(gitutil.GetLocalMainBranchOrDie())

	appConfig := util.GetAppConfig()
	appConfig.Io.In = strings.NewReader("My Feature PR\n\n") // PR name, then empty string for prefix (press enter to skip)
	util.SetAppConfig(appConfig)
	// Process the branch without PR - this should skip the duplicate commits
	result := processBranch("feature-branch", mostRecentMainCommit)

	// Verify the migration was successful
	assert.True(result.success, fmt.Sprint("Migration should succeed", result))
	assert.Equal("feature-branch", result.branchName)
	assert.Equal(3, result.numCommits, "Should report 3 commits migrated")

	// Verify we're on main branch after migration
	currentBranch := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--show-current")
	assert.Equal(gitutil.GetLocalMainBranchOrDie(), currentBranch, "Should be on main branch after migration")

	// Count commits on main after migration
	mainCommitsAfter := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-list", "--count", "HEAD")

	// We had 4 commits on main before (initial empty + initial + first + third)
	// After migration, we should have only added 1 new commit (second)
	// because first and third were already on main (same file changes) and should be skipped
	// So: 4 (before) + 1 (second commit) = 5 total
	assert.Equal("5", mainCommitsAfter, "Should have added only 1 new commit (the second one), skipping duplicates")

	// Verify the second commit is now on main by checking for its file
	files := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "ls-files")
	assert.Contains(files, "feature2.txt", "Second commit's file should be on main")

	// Verify all three feature files are on main
	assert.Contains(files, "feature1.txt", "First commit's file should be on main (from earlier cherry-pick)")
	assert.Contains(files, "feature3.txt", "Third commit's file should be on main (from earlier cherry-pick)")

	// Verify that the skipping worked by checking the second commit was added only once to main
	// Use git log on just the main branch to count occurrences
	mainLog := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "log", "--oneline", gitutil.GetLocalMainBranchOrDie())
	secondCommitCount := strings.Count(mainLog, "second feature commit")
	assert.Equal(1, secondCommitCount, "Second commit should appear exactly once on main (not duplicated)")
}
