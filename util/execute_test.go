package util

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
	// nolint:errcheck
	DefaultExecutor{}.Execute(
		ExecuteOptions{ShouldRetry: RetryUpTo(1)},
		"nonexistent-program-sd-test",
		"--body", "line1\nline2\nline3",
	)

	logOutput := logBuf.String()
	assert.Contains(logOutput, "Retrying")
	assert.Contains(logOutput, "nonexistent-program-sd-test --body line1")
	assert.NotContains(logOutput, "line2")
	assert.NotContains(logOutput, "line3")
}

func TestRetryUpTo(t *testing.T) {
	assert := assert.New(t)

	retry := RetryUpTo(2)
	assert.True(retry(1, ""))
	assert.True(retry(2, ""))
	assert.False(retry(3, ""))
}

func TestRetryOnIndexLock(t *testing.T) {
	assert := assert.New(t)

	const lockErr = "fatal: Unable to create '/repo/.git/index.lock': File exists."
	assert.True(RetryOnIndexLock(1, lockErr))
	assert.True(RetryOnIndexLock(IndexLockRetries, lockErr))
	// Once the execution count exceeds the limit, stop retrying.
	assert.False(RetryOnIndexLock(IndexLockRetries+1, lockErr))
	// Unrelated errors must not retry.
	assert.False(RetryOnIndexLock(1, "error: pathspec 'foo' did not match any file(s)"))
	assert.False(RetryOnIndexLock(1, "fatal: not a git repository"))
}

func TestDefaultShouldRetryByProgramName(t *testing.T) {
	assert := assert.New(t)

	const lockErr = "fatal: Unable to create '/repo/.git/index.lock': File exists."

	gitRetry := defaultShouldRetry("git")
	if assert.NotNil(gitRetry) {
		assert.True(gitRetry(1, lockErr))
		assert.False(gitRetry(1, "some other error"))
	}

	ghRetry := defaultShouldRetry("gh")
	if assert.NotNil(ghRetry) {
		assert.True(ghRetry(GhRetries, "any error"))
		assert.False(ghRetry(GhRetries+1, "any error"))
	}

	assert.Nil(defaultShouldRetry("nonexistent-program-sd-test"))
}

func TestGitIndexLockRetriedByDefault(t *testing.T) {
	assert := assert.New(t)

	// Capture log output.
	var logBuf bytes.Buffer
	handler := NewPrettyHandler(&logBuf, slog.HandlerOptions{Level: slog.LevelWarn})
	slog.SetDefault(slog.New(handler))

	// Count sleeps instead of waiting.
	var sleepCount int
	SetDefaultSleep(func(d time.Duration) { sleepCount++ })
	defer SetDefaultSleep(time.Sleep)

	// Create a real git repo with a held index.lock so "git add" always hits contention.
	dir := t.TempDir()
	executor := DefaultExecutor{}
	if _, err := executor.Execute(ExecuteOptions{}, "git", "-C", dir, "init"); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(dir, ".git", "index.lock")
	if err := os.WriteFile(lockPath, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatal(err)
		}
	}()

	// No explicit ShouldRetry: the "git" default (RetryOnIndexLock) must engage.
	// nolint:errcheck
	out, runErr := DefaultExecutor{}.Execute(ExecuteOptions{}, "git", "add", "-A")

	assert.Error(runErr)
	assert.Contains(out, "index.lock")
	// IndexLockRetries retries means that many sleeps and "Retrying" warnings.
	assert.Equal(IndexLockRetries, sleepCount)
	assert.Equal(IndexLockRetries, strings.Count(logBuf.String(), "Retrying"))
}

func TestNoRetryWhenShouldRetryNilAndNoDefault(t *testing.T) {
	assert := assert.New(t)

	var sleepCount int
	SetDefaultSleep(func(d time.Duration) { sleepCount++ })
	defer SetDefaultSleep(time.Sleep)

	// A non-git/gh command with no explicit predicate has no default retry.
	// nolint:errcheck
	_, err := DefaultExecutor{}.Execute(
		ExecuteOptions{},
		"nonexistent-program-sd-test",
	)

	assert.Error(err)
	assert.Equal(0, sleepCount)
}

func TestShouldRetrySeesStderr(t *testing.T) {
	assert := assert.New(t)

	SetDefaultSleep(func(d time.Duration) {})
	defer SetDefaultSleep(time.Sleep)

	const marker = "sd-test-stderr-marker"
	var seenStderr string
	// nolint:errcheck
	DefaultExecutor{}.Execute(
		ExecuteOptions{
			ShouldRetry: func(executionCount int, stderr string) bool {
				seenStderr = stderr
				return false
			},
		},
		"sh", "-c", "echo "+marker+" 1>&2; exit 1",
	)

	assert.Contains(seenStderr, marker)
}
