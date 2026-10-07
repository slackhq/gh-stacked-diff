package commands

import (
	"log/slog"
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestGetTargetCommits_WhenCommandLineCommitsNewestFirst_ReturnsEarliestFirst(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")
	indicator := string(templates.IndicatorTypeList)

	// List indicators are newest-first, so "1" is "third" and "3" is "first".
	targetCommits := getTargetCommits([]string{"1", "3", "2"}, &indicator, interactive.CommitSelectionOptions{})

	subjects := util.MapSlice(targetCommits, func(commit templates.GitLog) string {
		return commit.Subject
	})
	assert.Equal([]string{"first", "second", "third"}, subjects)
}
