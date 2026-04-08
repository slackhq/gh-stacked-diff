package util

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryLogOnlyIncludesFirstLineOfCommand(t *testing.T) {
	assert := assert.New(t)

	// Capture log output.
	var logBuf bytes.Buffer
	handler := NewPrettyHandler(&logBuf, slog.HandlerOptions{Level: slog.LevelWarn})
	slog.SetDefault(slog.New(handler))

	// Skip sleep during retries.
	SetDefaultSleep(func(d time.Duration) {})
	defer SetDefaultSleep(time.Sleep)

	// Use a command that doesn't exist so it always fails, with a multiline argument.
	DefaultExecutor{}.Execute(
		ExecuteOptions{Retries: 1},
		"nonexistent-program-sd-test",
		"--body", "line1\nline2\nline3",
	)

	logOutput := logBuf.String()
	assert.Contains(logOutput, "Retrying")
	assert.Contains(logOutput, "nonexistent-program-sd-test --body line1")
	assert.NotContains(logOutput, "line2")
	assert.NotContains(logOutput, "line3")
}
