package commands

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func TestSdCodeOwners_OutputsOwnersOfChangedFiles(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "first-not-changed")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetLocalMainBranchOrDie())

	testutil.AddCommit("second", "second-changed")

	testutil.AddCommit("third", "third-changed")

	codeOwners := "first-not-changed firstOwners\n" +
		"second-changed secondOwners\n" +
		"third-changed thirdOwners\n"

	util.ExecuteOrDie(util.ExecuteOptions{}, "mkdir", "-p", ".github")
	if writeErr := os.WriteFile(".github/CODEOWNERS", []byte(codeOwners), os.ModePerm); writeErr != nil {
		panic(writeErr)
	}
	out := testParseArguments("code-owners")

	assert.Equal("Owner: secondOwners\n"+
		"second-changed\n"+
		"\n"+
		"Owner: thirdOwners\n"+
		"third-changed\n", out)
}
