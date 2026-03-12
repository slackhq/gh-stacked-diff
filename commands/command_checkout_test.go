package commands

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinyspeck/gh-stacked-diff/v2/templates"
	"github.com/tinyspeck/gh-stacked-diff/v2/testutil"
	"github.com/tinyspeck/gh-stacked-diff/v2/util"
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
