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

func TestGetGhStackLabels_LabelsFromBottomAndSharesLabelWithinStack(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	writeStackFile(t, twoStacksJson)

	// Listed newest first: d (stack 2), other (none), c and a (stack 1).
	assert.Equal(t, []GhStackLabel{{true, "B"}, {}, {true, "A"}, {true, "A"}}, GetGhStackLabels([]string{"d", "other", "c", "a"}))
	assert.Equal(t, []GhStackLabel{{}, {}}, GetGhStackLabels([]string{"x", "y"}))
}

func TestGetGhStackLabels_WhenOnlyOneStack_OmitsLabel(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	writeStackFile(t, twoStacksJson)

	assert.Equal(t, []GhStackLabel{{}, {true, ""}, {true, ""}}, GetGhStackLabels([]string{"other", "c", "a"}))
}

func TestStackLabel_ContinuesLikeSpreadsheetColumns(t *testing.T) {
	assert.Equal(t, "A", stackLabel(0))
	assert.Equal(t, "Z", stackLabel(25))
	assert.Equal(t, "AA", stackLabel(26))
	assert.Equal(t, "AB", stackLabel(27))
	assert.Equal(t, "ZZ", stackLabel(701))
	assert.Equal(t, "AAA", stackLabel(702))
}
