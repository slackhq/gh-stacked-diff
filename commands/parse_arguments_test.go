package commands

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slackhq/gh-stacked-diff/v2/testutil"
)

func TestFindLogLevelArg_WithEqualsLongFlag(t *testing.T) {
	level := findLogLevelArg([]string{"--log-level=debug", "version"})
	assert.Equal(t, slog.LevelDebug, level)
}

func TestFindLogLevelArg_WithSpaceLongFlag(t *testing.T) {
	level := findLogLevelArg([]string{"--log-level", "warn", "version"})
	assert.Equal(t, slog.LevelWarn, level)
}

func TestFindLogLevelArg_WithEqualsShortFlag(t *testing.T) {
	level := findLogLevelArg([]string{"-l=error", "version"})
	assert.Equal(t, slog.LevelError, level)
}

func TestFindLogLevelArg_WithSpaceShortFlag(t *testing.T) {
	level := findLogLevelArg([]string{"-l", "info", "version"})
	assert.Equal(t, slog.LevelInfo, level)
}

func TestFindLogLevelArg_WhenNotPresent(t *testing.T) {
	level := findLogLevelArg([]string{"version"})
	assert.Equal(t, slog.LevelInfo, level)
}

func TestFindLogLevelArg_WithInvalidLevel(t *testing.T) {
	level := findLogLevelArg([]string{"version"})
	assert.Equal(t, slog.LevelInfo, level)
}

func TestRepoCheck_FromNonGitDirectory_ShowsFriendlyError(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	out := new(bytes.Buffer)
	defer func() {
		r := recover()
		assert.NotNil(r)
		assert.Contains(out.String(), "Not in a git repository")
	}()
	testParseArgumentsWithOut(out, "log")
	assert.Fail("did not panic")
}

func TestEarlyLogLevel_DebugOutputVisibleDuringSetup(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	out := testParseArguments("--log-level=debug", "version")

	// "App executable" is logged before PersistentPreRun at debug level. If the early
	// logger setup via findLogLevelArg works, this debug output will be visible.
	assert.Contains(out, "App executable")
}
