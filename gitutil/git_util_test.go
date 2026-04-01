package gitutil

import (
	"log/slog"
	"os"
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

// setupWorktreeTestRepo creates a git repo with a remote, an initial commit,
// and changes into the local-repo directory. Cached branch values are reset.
// This is a lightweight alternative to testutil.InitTest which cannot be used
// from gitutil tests due to a transitive import cycle
// (gitutil test → testutil → interactive → gitutil).
func setupWorktreeTestRepo(t *testing.T) {
	t.Helper()
	handler := util.NewPrettyHandler(os.Stdout, slog.HandlerOptions{Level: slog.LevelError})
	slog.SetDefault(slog.New(handler))
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ResetCacheForTesting()
	})
	ResetCacheForTesting()
	util.SetGlobalExecutor(util.DefaultExecutor{})

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "init", "--bare", "remote-repo")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "clone", "remote-repo", "local-repo")
	if err := os.Chdir("local-repo"); err != nil {
		t.Fatal(err)
	}
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "config", "user.email", "test@example.com")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "config", "user.name", "Test")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "--allow-empty", "-m", "initial commit")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetCurrentBranchName())
	// Set origin/HEAD so GetRemoteMainBranch works.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "remote", "set-head", "origin", util.GetCurrentBranchName())
}

func TestGetSecondaryWorktreeBranch_WhenNoWorktrees_ReturnsEmpty(t *testing.T) {
	setupWorktreeTestRepo(t)

	result := getSecondaryWorktreeBranch()

	assert.Equal(t, "", result)
}

func TestGetSecondaryWorktreeBranch_WhenInMainWorktree_ReturnsEmpty(t *testing.T) {
	setupWorktreeTestRepo(t)

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "feature-branch")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../feature-worktree", "feature-branch")
	t.Cleanup(func() {
		_, _ = util.Execute(util.ExecuteOptions{}, "git", "worktree", "remove", "--force", "../feature-worktree")
	})

	result := getSecondaryWorktreeBranch()

	assert.Equal(t, "", result)
}

func TestGetSecondaryWorktreeBranch_WhenInSecondaryWorktree_ReturnsBranch(t *testing.T) {
	setupWorktreeTestRepo(t)

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "feature-branch")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../feature-worktree", "feature-branch")

	if err := os.Chdir("../feature-worktree"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(t.TempDir())
	})

	result := getSecondaryWorktreeBranch()

	assert.Equal(t, "feature-branch", result)
}

func TestGetLocalMainBranch_WhenInMainWorktree_ReturnsRemoteMainBranch(t *testing.T) {
	setupWorktreeTestRepo(t)

	branch, err := GetLocalMainBranch()

	assert.NoError(t, err)
	remoteBranch, remoteErr := GetRemoteMainBranch()
	assert.NoError(t, remoteErr)
	assert.Equal(t, remoteBranch, branch)
}

func TestGetLocalMainBranch_WhenInSecondaryWorktree_ReturnsWorktreeBranch(t *testing.T) {
	setupWorktreeTestRepo(t)

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "feature-branch")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../feature-worktree", "feature-branch")

	if err := os.Chdir("../feature-worktree"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(t.TempDir())
	})

	branch, err := GetLocalMainBranch()

	assert.NoError(t, err)
	assert.Equal(t, "feature-branch", branch)
}

func TestGetLocalMainBranch_CachesResult(t *testing.T) {
	setupWorktreeTestRepo(t)

	branch1, err1 := GetLocalMainBranch()
	branch2, err2 := GetLocalMainBranch()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, branch1, branch2)
}

func TestGetRemoteMainBranch_WhenInSecondaryWorktree_StillReturnsRemoteMain(t *testing.T) {
	setupWorktreeTestRepo(t)

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "feature-branch")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../feature-worktree", "feature-branch")

	if err := os.Chdir("../feature-worktree"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(t.TempDir())
	})

	remoteBranch, err := GetRemoteMainBranch()

	assert.NoError(t, err)
	assert.NotEqual(t, "feature-branch", remoteBranch)
	assert.Contains(t, []string{"main", "master"}, remoteBranch)
}
