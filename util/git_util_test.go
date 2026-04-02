package util

import (
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupWorktreeTestRepo creates a git repo with a remote, an initial commit,
// and changes into the local-repo directory. Cached branch values are reset.
func setupWorktreeTestRepo(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	// Save and restore cached state.
	oldRemoteMainBranch := remoteMainBranch
	oldLocalMainBranch := localMainBranch
	oldRemoteMainBranchOnce := remoteMainBranchOnce
	oldLocalMainBranchOnce := localMainBranchOnce
	oldMainBranchNameForHelp := mainBranchNameForHelp
	oldExecutor := globalExecutor
	t.Cleanup(func() {
		remoteMainBranch = oldRemoteMainBranch
		localMainBranch = oldLocalMainBranch
		remoteMainBranchOnce = oldRemoteMainBranchOnce
		localMainBranchOnce = oldLocalMainBranchOnce
		mainBranchNameForHelp = oldMainBranchNameForHelp
		globalExecutor = oldExecutor
	})
	// Reset caches for this test.
	remoteMainBranch = ""
	localMainBranch = ""
	remoteMainBranchOnce = new(sync.Once)
	localMainBranchOnce = new(sync.Once)
	mainBranchNameForHelp = ""
	globalExecutor = DefaultExecutor{}

	ExecuteOrDie(ExecuteOptions{}, "git", "init", "--bare", "remote-repo")
	ExecuteOrDie(ExecuteOptions{}, "git", "clone", "remote-repo", "local-repo")
	if err := os.Chdir("local-repo"); err != nil {
		t.Fatal(err)
	}
	ExecuteOrDie(ExecuteOptions{}, "git", "config", "user.email", "test@example.com")
	ExecuteOrDie(ExecuteOptions{}, "git", "config", "user.name", "Test")
	ExecuteOrDie(ExecuteOptions{}, "git", "commit", "--allow-empty", "-m", "initial commit")
	ExecuteOrDie(ExecuteOptions{}, "git", "push", "origin", GetCurrentBranchName())
	// Set origin/HEAD so GetRemoteMainBranch works.
	ExecuteOrDie(ExecuteOptions{}, "git", "remote", "set-head", "origin", GetCurrentBranchName())
}

func TestGetSecondaryWorktreeBranch_WhenNoWorktrees_ReturnsEmpty(t *testing.T) {
	setupWorktreeTestRepo(t)

	result := getSecondaryWorktreeBranch()

	assert.Equal(t, "", result)
}

func TestGetSecondaryWorktreeBranch_WhenInMainWorktree_ReturnsEmpty(t *testing.T) {
	setupWorktreeTestRepo(t)

	ExecuteOrDie(ExecuteOptions{}, "git", "branch", "feature-branch")
	ExecuteOrDie(ExecuteOptions{}, "git", "worktree", "add", "../feature-worktree", "feature-branch")
	t.Cleanup(func() {
		_, _ = Execute(ExecuteOptions{}, "git", "worktree", "remove", "--force", "../feature-worktree")
	})

	// We're still in the main worktree (local-repo).
	result := getSecondaryWorktreeBranch()

	assert.Equal(t, "", result)
}

func TestGetSecondaryWorktreeBranch_WhenInSecondaryWorktree_ReturnsBranch(t *testing.T) {
	setupWorktreeTestRepo(t)

	ExecuteOrDie(ExecuteOptions{}, "git", "branch", "feature-branch")
	ExecuteOrDie(ExecuteOptions{}, "git", "worktree", "add", "../feature-worktree", "feature-branch")

	if err := os.Chdir("../feature-worktree"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Must leave the worktree directory before removing it.
		_ = os.Chdir(t.TempDir())
	})

	result := getSecondaryWorktreeBranch()

	assert.Equal(t, "feature-branch", result)
}

func TestGetLocalMainBranch_WhenInMainWorktree_ReturnsRemoteMainBranch(t *testing.T) {
	setupWorktreeTestRepo(t)

	branch, err := GetLocalMainBranch()

	assert.NoError(t, err)
	// In the main worktree, local main should be the remote main branch name.
	remoteBranch, remoteErr := GetRemoteMainBranch()
	assert.NoError(t, remoteErr)
	assert.Equal(t, remoteBranch, branch)
}

func TestGetLocalMainBranch_WhenInSecondaryWorktree_ReturnsWorktreeBranch(t *testing.T) {
	setupWorktreeTestRepo(t)

	ExecuteOrDie(ExecuteOptions{}, "git", "branch", "feature-branch")
	ExecuteOrDie(ExecuteOptions{}, "git", "worktree", "add", "../feature-worktree", "feature-branch")

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

	ExecuteOrDie(ExecuteOptions{}, "git", "branch", "feature-branch")
	ExecuteOrDie(ExecuteOptions{}, "git", "worktree", "add", "../feature-worktree", "feature-branch")

	if err := os.Chdir("../feature-worktree"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(t.TempDir())
	})

	// Remote main should still be the actual remote main (e.g. "main"), not the worktree branch.
	remoteBranch, err := GetRemoteMainBranch()

	assert.NoError(t, err)
	assert.NotEqual(t, "feature-branch", remoteBranch)
	assert.Contains(t, []string{"main", "master"}, remoteBranch)
}
