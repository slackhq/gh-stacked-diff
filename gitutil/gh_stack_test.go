package gitutil

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

const twoStacksJson = `{"schemaVersion":1,"repository":"github.com:x/y","stacks":[
 {"trunk":{"branch":"main"},"branches":[{"branch":"a"},{"branch":"b"},{"branch":"c"}]},
 {"trunk":{"branch":"main"},"branches":[{"branch":"d"}]}]}`

func writeStackFile(t *testing.T, contents string) {
	gitDir := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	assert.NoError(t, os.WriteFile(filepath.Join(gitDir, "gh-stack"), []byte(contents), 0o644))
}

func TestGetGhStackBranches_WhenNoStackFile_ReturnsEmpty(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)

	assert.Equal(t, []string{}, GetGhStackBranches("a"))
}

func TestGetGhStackBranches_WhenBranchInStack_ReturnsWholeStackInOrder(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	writeStackFile(t, twoStacksJson)

	assert.Equal(t, []string{"a", "b", "c"}, GetGhStackBranches("b"))
	assert.Equal(t, []string{"d"}, GetGhStackBranches("d"))
}

func TestGetGhStackBranches_WhenBranchNotInStack_ReturnsEmpty(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	writeStackFile(t, twoStacksJson)

	assert.Equal(t, []string{}, GetGhStackBranches("other"))
}

func TestGetGhStackBranches_WhenInvalidJson_ReturnsEmpty(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	writeStackFile(t, "not json")

	assert.Equal(t, []string{}, GetGhStackBranches("a"))
}

func TestGetGhStackBranches_WhenInSecondaryWorktree_ReadsStackFileFromCommonDir(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	writeStackFile(t, twoStacksJson)
	testutil.SetupSecondaryWorktree(t)

	assert.Equal(t, []string{"a", "b", "c"}, GetGhStackBranches("a"))
}
