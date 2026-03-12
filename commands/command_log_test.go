package commands

import (
	"log/slog"
	"testing"

	"github.com/fatih/color"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"

	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestSdLog_WhenRemoteHasSomeCommits_PrintsNewLogsOnly(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())

	testutil.AddCommit("second", "")

	out := testParseArguments("log")

	assert.NotContains(out, "first")
	assert.Contains(out, "second")
}

func TestSdLog_WhenPrCreatedForSomeCommits_PrintsCheckForCommitsWithPrs(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	out := testParseArguments("log")

	assert.Contains(out, "✅")
}

func TestSdLog_WhenNotOnMain_OnlyShowsCommitsNotOnMain(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "-b", "my-branch")

	testutil.AddCommit("second", "")

	out := testParseArguments("log")

	assert.NotContains(out, "first")
	assert.Contains(out, "second")
}

func TestSdLog_WhenCommitHasBranch_PrintsExtraBranchCommits(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")
	testutil.AddCommit("forth", "")

	testParseArguments("update", "4", "3", "2", "1")

	out := testParseArguments("log")

	allCommits := templates.GetAllCommits()

	assert.Equal("1. ✅ "+color.YellowString(allCommits[0].Commit)+" first\n"+
		"      - second\n"+
		"      - third\n"+
		"      - forth\n",
		out)
}

func TestSdLog_WhenBranchHasManyCommits_PrintsLatest3(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")
	testutil.AddCommit("forth", "")
	testutil.AddCommit("fifth", "")
	testutil.AddCommit("sixth", "")

	testParseArguments("update", "6", "5", "4", "3", "2", "1")

	out := testParseArguments("log")

	allCommits := templates.GetAllCommits()
	assert.Equal("1. ✅ "+color.YellowString(allCommits[0].Commit)+" first\n"+
		"      - [hiding 3 previous...]\n"+
		"      - fifth\n"+
		"      - sixth\n",
		out)
}

func TestSdLog_LogsOutput(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	out := testParseArguments("log")

	assert.Contains(out, "first")
}

func TestSdLog_WhenManyCommits_PadsFirstCommits(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")
	testutil.AddCommit("fourth", "")
	testutil.AddCommit("fifth", "")
	testutil.AddCommit("sixth", "")
	testutil.AddCommit("seventh", "")
	testutil.AddCommit("eigth", "")
	testutil.AddCommit("ninth", "")
	testutil.AddCommit("tenth", "")

	out := testParseArguments("log")

	assert.Contains(out, "\n 2.    ")
	assert.Contains(out, "\n10.    ")
}

func TestSdLog_WhenMultiplePrs_MatchesAllPrs(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")

	testParseArguments("new", "2")
	testParseArguments("new", "1")

	out := testParseArguments("log")

	assert.Regexp("✅.*first", out)
	assert.Regexp("✅.*second", out)
}
