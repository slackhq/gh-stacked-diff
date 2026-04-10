package commands

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func TestSdReplaceConflicts_WhenConflictOnLastCommit_ReplacesCommit(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "file-with-conflicts")
	testutil.CommitFileChange("second", "file-with-conflicts", "1")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())
	allCommits := templates.GetAllCommits()
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", allCommits[1].Commit)
	testutil.CommitFileChange("third", "file-with-conflicts", "2")

	testParseArguments("new", "1")

	testParseArguments("checkout", "1")

	_, mergeErr := util.Execute(util.ExecuteOptions{}, "git", "merge", "origin/"+gitutil.GetRemoteMainBranchOrDie())
	assert.NotNil(mergeErr)

	if writeErr := os.WriteFile("file-with-conflicts", []byte("1\n2"), 0644); writeErr != nil {
		panic(writeErr)
	}
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")

	continueOptions := util.ExecuteOptions{EnvironmentVariables: []string{"GIT_EDITOR=true"}}
	util.ExecuteOrDie(continueOptions, "git", "merge", "--continue")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", gitutil.GetLocalMainBranchOrDie())

	_, rebaseErr := util.Execute(util.ExecuteOptions{}, "git", "rebase", "origin/"+gitutil.GetRemoteMainBranchOrDie())
	assert.NotNil(rebaseErr)

	testParseArguments("replace-conflicts", "--confirm=true")

	allCommits = templates.GetAllCommits()

	assert.Equal(4, len(allCommits))
	assert.Equal("third", allCommits[0].Subject)
	assert.Equal("second", allCommits[1].Subject)
	assert.Equal("first", allCommits[2].Subject)
	assert.Equal(testutil.InitialCommitSubject, allCommits[3].Subject)

	dirEntries, err := os.ReadDir(".")
	if err != nil {
		panic("Could not read dir: " + err.Error())
	}
	assert.Equal(2, len(dirEntries))
	assert.Equal(".git", dirEntries[0].Name())
	assert.Equal("file-with-conflicts", dirEntries[1].Name())

	contents, readErr := os.ReadFile("file-with-conflicts")
	assert.Nil(readErr)
	// Add a .? to account for eol on windows.
	assert.Regexp("1.?\n2", string(contents))
}
