package commands

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func TestSdWaitForMerge_WaitsForMerge(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	allCommits := templates.GetAllCommits()
	testExecutor.SetResponse("2025-01-01", nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	out := testParseArguments("--log-level=info", "wait-for-merge", allCommits[0].Commit)

	assert.Contains(out, "Merged!")
}
