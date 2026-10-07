package templates

import (
	"log/slog"
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestSortEarliestFirst_WhenNewestFirst_ReturnsEarliestFirst(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")
	newCommits := GetNewCommits("HEAD", "")

	// New commits are newest-first, so shuffle them.
	sorted := SortEarliestFirst([]GitLog{newCommits[0], newCommits[2], newCommits[1]}, "")

	subjects := util.MapSlice(sorted, func(commit GitLog) string {
		return commit.Subject
	})
	assert.Equal([]string{"first", "second", "third"}, subjects)
}

func TestSortEarliestFirst_WhenCommitNotNew_Panics(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")

	assert.PanicsWithValue(
		"Commit abc1234 does not exist on "+gitutil.GetLocalMainBranchOrDie()+". Check `sd log` for available commits.",
		func() { SortEarliestFirst([]GitLog{{Commit: "abc1234"}}, "") },
	)
}
