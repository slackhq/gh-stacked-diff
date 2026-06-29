package gitutil

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

// Reproduces the real-world bug: a multi-commit cherry-pick is interrupted by
// ".git/index.lock" contention (e.g. an IDE running git in the background) *after*
// the first commit has already been applied. A verbatim retry of the whole
// cherry-pick fails with "cherry-pick is already in progress", so the operation
// must abort the partially-applied cherry-pick and restart from a clean state.
func TestCherryPick_WhenIndexLockInterruptsMultiCommit_AbortsRestartsAndSucceeds(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	mainBranch := GetLocalMainBranchOrDie()

	// Two commits on a feature branch to cherry-pick back onto main. Distinct files
	// so the cherry-pick itself does not conflict.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", "-c", "feat")
	testutil.CommitFileChange("pick A", "fa", "A")
	testutil.CommitFileChange("pick B", "fb", "B")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", mainBranch)

	gitDir := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "--absolute-git-dir")
	lockPath := filepath.Join(gitDir, "index.lock")
	hookPath := filepath.Join(gitDir, "hooks", "post-commit")
	markerPath := filepath.Join(gitDir, "sd-test-lock-fired")

	// After the first cherry-picked commit lands, create index.lock exactly once so the
	// second pick of the same invocation is interrupted mid-sequence.
	hook := "#!/bin/sh\n" +
		"if [ ! -f '" + markerPath + "' ]; then\n" +
		"  : > '" + markerPath + "'\n" +
		"  : > '" + lockPath + "'\n" +
		"fi\n"
	if err := os.WriteFile(hookPath, []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}

	// Simulate the contending process releasing the lock during backoff: remove the
	// lock whenever the retry sleeps.
	util.SetDefaultSleep(func(d time.Duration) { _ = os.Remove(lockPath) })
	defer util.SetDefaultSleep(time.Sleep)

	_, err := CherryPick(util.ExecuteOptions{}, "feat~1", "feat")

	assert.NoError(err)
	logOut := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "--pretty=format:%s")
	// Both commits applied exactly once (idempotent restart, not doubled).
	assert.Equal(1, strings.Count(logOut, "pick A"))
	assert.Equal(1, strings.Count(logOut, "pick B"))
	// No cherry-pick left in progress.
	assert.NoFileExists(filepath.Join(gitDir, "CHERRY_PICK_HEAD"))
}

// A merge conflict is not index.lock contention: CherryPick must return the error
// to the caller without aborting, so the caller can run its own conflict recovery.
func TestCherryPick_WhenConflict_ReturnsErrorWithoutAborting(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	mainBranch := GetLocalMainBranchOrDie()

	// Create a commit on a branch that conflicts with a divergent change on main.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", "-c", "feat")
	testutil.CommitFileChange("feat change", "shared", "feat version")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", mainBranch)
	testutil.CommitFileChange("main change", "shared", "main version")

	_, err := CherryPick(util.ExecuteOptions{}, "feat")

	assert.Error(err)
	gitDir := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "--absolute-git-dir")
	// The conflicted cherry-pick is left in progress for the caller to recover.
	assert.FileExists(filepath.Join(gitDir, "CHERRY_PICK_HEAD"))
}
