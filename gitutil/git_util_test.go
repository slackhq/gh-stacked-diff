package gitutil

import (
	"log/slog"
	"os"
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestGetSecondaryWorktreeBranch_WhenNoWorktrees_ReturnsEmpty(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)

	result := getSecondaryWorktreeBranch()

	assert.Equal(t, "", result)
}

func TestGetSecondaryWorktreeBranch_WhenInMainWorktree_ReturnsEmpty(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../feature-worktree")

	result := getSecondaryWorktreeBranch()

	assert.Equal(t, "", result)
}

func TestGetSecondaryWorktreeBranch_WhenInSecondaryWorktree_ReturnsBranch(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	testutil.SetupSecondaryWorktree(t)

	result := getSecondaryWorktreeBranch()

	assert.Equal(t, "secondary-worktree", result)
}

func TestGetSecondaryWorktreeBranch_WhenInSecondaryWorktree_ReturnsLastPathComponent(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../my-custom-dir")

	if err := os.Chdir("../my-custom-dir"); err != nil {
		t.Fatal(err)
	}
	result := getSecondaryWorktreeBranch()

	assert.Equal(t, "my-custom-dir", result)
}

func TestGetSecondaryWorktreeBranch_WhenGuardNone_ReturnsBranchName(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	util.SetUserConfig(util.UserConfig{WorktreeMainBranchGuard: util.WorktreeMainBranchGuardNone})
	testutil.SetupSecondaryWorktree(t)

	result := getSecondaryWorktreeBranch()

	assert.Equal(t, "secondary-branch", result)
}

func TestGetWorktrees_WhenDetachedHead_IncludesCommitHash(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Create a worktree in detached HEAD state.
	commitHash := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "--short", "HEAD")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "--detach", "../detached-worktree")

	worktrees := GetWorktrees()

	assert.Equal(2, len(worktrees))
	assert.Equal(commitHash, worktrees[1].BranchOrCommit)
}

func TestGetMainWorktreePath_ReturnsMainWorktreePath(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	mainPath := testutil.SetupSecondaryWorktree(t)

	result := GetMainWorktreePath()

	assert.Equal(t, mainPath, result)
}

func TestGetMainWorktreePath_WhenNoWorktrees_ReturnsCurrentRepo(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)

	expected, _ := os.Getwd()
	result := GetMainWorktreePath()

	assert.Equal(t, expected, result)
}

func TestIsSecondaryWorktree_WhenInMainWorktree_ReturnsFalse(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)

	assert.False(t, IsSecondaryWorktree())
}

func TestIsSecondaryWorktree_WhenInSecondaryWorktree_ReturnsTrue(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	testutil.SetupSecondaryWorktree(t)

	assert.True(t, IsSecondaryWorktree())
}

func TestGetLocalMainBranch_WhenInMainWorktree_ReturnsRemoteMainBranch(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)

	branch, err := getLocalMainBranch()

	assert.NoError(t, err)
	remoteBranch, remoteErr := getRemoteMainBranch()
	assert.NoError(t, remoteErr)
	assert.Equal(t, remoteBranch, branch)
}

func TestGetLocalMainBranch_WhenInSecondaryWorktree_ReturnsWorktreeBranch(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	testutil.SetupSecondaryWorktree(t)

	branch, err := getLocalMainBranch()

	assert.NoError(t, err)
	assert.Equal(t, "secondary-worktree", branch)
}

func TestGetLocalMainBranch_CachesResult(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)

	branch1, err1 := getLocalMainBranch()
	branch2, err2 := getLocalMainBranch()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, branch1, branch2)
}

func TestGetRemoteMainBranch_WhenInSecondaryWorktree_StillReturnsRemoteMain(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	testutil.SetupSecondaryWorktree(t)

	remoteBranch, err := getRemoteMainBranch()

	assert.NoError(t, err)
	assert.NotEqual(t, "secondary-branch", remoteBranch)
	assert.Contains(t, []string{"main", "master"}, remoteBranch)
}

func TestGitPush_WhenLfsError_PushesLfsAndRetries(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelDebug)
	testutil.AddCommit("test commit", "")
	branch := util.GetCurrentBranchName()

	// Install a server-side hook that fails with an LFS error on the first push attempt.
	hookDir := "../remote-repo/hooks"
	// nolint:errcheck
	os.MkdirAll(hookDir, os.ModePerm)
	markerFile := hookDir + "/lfs-marker"
	hookScript := "#!/bin/sh\n" +
		"if [ ! -f \"" + markerFile + "\" ]; then\n" +
		"  touch \"" + markerFile + "\"\n" +
		"  echo \"remote: error: GH008: Your push referenced at least 1 unknown Git LFS objects:\" >&2\n" +
		"  echo \"remote: Try to push them with 'git lfs push --all'.\" >&2\n" +
		"  exit 1\n" +
		"fi\n" +
		"exit 0\n"
	// nolint:errcheck
	os.WriteFile(hookDir+"/pre-receive", []byte(hookScript), 0755)

	out, err := GitPush(util.ExecuteOptions{}, "push", "origin", branch)

	assert.NoError(err)
	assert.NotContains(out, "GH008")
}

func TestGitPushOrDie_WhenPersistentError_Panics(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelDebug)
	testutil.AddCommit("test commit", "")
	branch := util.GetCurrentBranchName()

	// Install a hook that always fails with a non-LFS error.
	hookDir := "../remote-repo/hooks"
	// nolint:errcheck
	os.MkdirAll(hookDir, os.ModePerm)
	hookScript := "#!/bin/sh\necho \"remote: error: some other error\" >&2\nexit 1\n"
	// nolint:errcheck
	os.WriteFile(hookDir+"/pre-receive", []byte(hookScript), 0755)

	assert.Panics(func() {
		GitPushOrDie(util.ExecuteOptions{}, "push", "origin", branch)
	})
}

func TestExtractRemoteFromPushArgs(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("origin", extractRemoteFromPushArgs([]string{"push", "origin", "main:main"}))
	assert.Equal("upstream", extractRemoteFromPushArgs([]string{"push", "upstream", "branch:branch"}))
	assert.Equal("origin", extractRemoteFromPushArgs([]string{"-c", "push.default=current", "push", "--force-with-lease", "-u"}))
	assert.Equal("origin", extractRemoteFromPushArgs([]string{"push", "--force-with-lease", "origin", "branch:branch"}))
}
