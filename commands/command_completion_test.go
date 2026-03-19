package commands

import (
	"log/slog"
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
