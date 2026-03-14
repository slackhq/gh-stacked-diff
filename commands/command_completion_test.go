package commands

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompletion_Zsh(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	output := testParseArguments("completion", "--shell", "zsh")
	assert := assert.New(t)
	assert.Contains(output, "#compdef sd")
	assert.Contains(output, "_sd")
	// Verify known commands appear in the completion.
	assert.Contains(output, "update")
	assert.Contains(output, "log")
	assert.Contains(output, "new")
	assert.Contains(output, "version")
}

func TestCompletion_MissingShellFlag(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	assert.PanicsWithValue(t, "Panicking instead of exiting with code 1", func() {
		testParseArguments("completion")
	})
}

func TestCompletion_Bash(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	output := testParseArguments("completion", "--shell", "bash")
	assert := assert.New(t)
	assert.Contains(output, "complete -F _sd sd")
	assert.Contains(output, "_sd()")
	assert.Contains(output, "update")
	assert.Contains(output, "log")
}

func TestCompletion_Fish(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	output := testParseArguments("completion", "--shell", "fish")
	assert := assert.New(t)
	assert.Contains(output, "complete -c sd")
	assert.Contains(output, "__fish_use_subcommand")
	assert.Contains(output, "update")
	assert.Contains(output, "log")
	// Verify single-dash flags use -o (old-style) not -l (long/double-dash).
	assert.Contains(output, "-o log-level")
	assert.NotContains(output, "-l log-level")
}

func TestCompletion_Powershell(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	output := testParseArguments("completion", "--shell", "powershell")
	assert := assert.New(t)
	assert.Contains(output, "Register-ArgumentCompleter")
	assert.Contains(output, "CompletionResult")
	assert.Contains(output, "update")
	assert.Contains(output, "log")
}

func TestCompletion_UnsupportedShell(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	assert.PanicsWithValue(t, "Panicking instead of exiting with code 1", func() {
		testParseArguments("completion", "--shell", "nushell")
	})
}

func TestCompletion_TooManyArgs(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	assert.PanicsWithValue(t, "Panicking instead of exiting with code 1", func() {
		testParseArguments("completion", "extra")
	})
}
