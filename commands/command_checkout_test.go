package commands

import (
	"log/slog"
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestSdCheckout_ChecksOutBranch(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	allCommits := templates.GetAllCommits()

	testParseArguments("new", "1")

	testParseArguments("checkout", allCommits[0].Commit)

	assert.Equal(allCommits[0].Branch, util.GetCurrentBranchName())
}

func TestSdCheckout_ListIndicatorUsesLocalMain(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	allCommits := templates.GetAllCommits()
	testParseArguments("new", "1")
	testParseArguments("new", "2")

	// Move to a branch that does not contain the local main commits.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "other", "HEAD~2")

	// List index 2 is the oldest commit on local main ("first").
	testParseArguments("checkout", "2")

	assert.Equal(allCommits[1].Branch, util.GetCurrentBranchName())
}
