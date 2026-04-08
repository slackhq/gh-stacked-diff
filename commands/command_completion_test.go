package commands

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slackhq/gh-stacked-diff/v2/testutil"
)

func TestSdCompletion_Bash(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	out := testParseArguments("completion", "bash")

	assert.Contains(out, "bash completion V2")
}

func TestSdCompletion_Zsh(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	out := testParseArguments("completion", "zsh")

	assert.Contains(out, "zsh completion")
}

func TestSdCompletion_Fish(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	out := testParseArguments("completion", "fish")

	assert.Contains(out, "fish completion")
}

func TestSdCompletion_FromNonGitDirectory(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelDebug)
	// Move to a non-git directory to verify completion works without a repo.
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	out := testParseArguments("completion", "bash")
	assert.Contains(out, "bash completion V2")
}
