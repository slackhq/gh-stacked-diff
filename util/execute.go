package util

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// lockedBuffer is a [bytes.Buffer] safe for concurrent writes, needed because the
// stdout and stderr copy goroutines started by [exec.Cmd] can both write to it.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// ShouldRetryFunc decides whether to re-run a command after a failure.
// It receives the 1-based execution count (1 = first attempt) and the command's stderr.
// Return true to retry.
type ShouldRetryFunc func(executionCount int, stderr string) bool

// Options for [ExecuteWithOptions].
type ExecuteOptions struct {
	// What to use for input and output. Overriding input is useful for "git apply"
	// If output is not set then output is returned from Execute.
	// Any nil In/Err/Out values are ignored.
	Io StdIo
	// For example "MY_VAR=some_value"
	EnvironmentVariables []string
	// ShouldRetry decides whether to re-run after a failure. If nil, a default is
	// chosen based on the program name (see [defaultShouldRetry]).
	ShouldRetry ShouldRetryFunc
}

// defaultShouldRetry returns the retry policy used when [ExecuteOptions.ShouldRetry] is nil.
// "git" commands retry on ".git/index.lock" contention, "gh" commands retry on any error.
func defaultShouldRetry(programName string) ShouldRetryFunc {
	switch programName {
	case "git":
		return RetryOnIndexLock
	case "gh":
		return RetryUpTo(GhRetries)
	default:
		return nil
	}
}

// Provides a simple way to execute shell commands.
// Allows swapping in a [TestExecutor] via Dependency Injection during tests.
type Executor interface {
	Execute(options ExecuteOptions, programName string, args ...any) (string, error)
}

func flattenArgs(args []any) []string {
	var flat []string
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			flat = append(flat, v)
		case []string:
			flat = append(flat, v...)
		default:
			panic(fmt.Sprintf("Execute args must be string or []string, got %T", arg))
		}
	}
	return flat
}

var globalExecutor Executor = DefaultExecutor{}

// Default implementation of [Executor].
type DefaultExecutor struct{}

// Sets the executor that [Execute] will use.
func SetGlobalExecutor(executor Executor) {
	globalExecutor = executor
}

// Implementation of Execute that uses [exec.Command].
func (defaultExecutor DefaultExecutor) Execute(options ExecuteOptions, programName string, args ...any) (string, error) {
	flatArgs := flattenArgs(args)
	retry := options.ShouldRetry
	if retry == nil {
		retry = defaultShouldRetry(programName)
	}
	executionCount := 0
	for {
		executionCount++
		cmd := exec.Command(programName, flatArgs...)
		if options.EnvironmentVariables != nil {
			cmd.Env = append(os.Environ(), options.EnvironmentVariables...)
		}
		if options.Io.In != nil {
			cmd.Stdin = options.Io.In
		}
		// stderr is captured separately (in addition to the combined output) so that
		// the retry predicate can inspect it. combined is written by both the stdout and
		// stderr copy goroutines, so it must be synchronized.
		combined := &lockedBuffer{}
		var stderrBuf bytes.Buffer
		if options.Io.Out != nil {
			cmd.Stdout = options.Io.Out
		} else {
			cmd.Stdout = combined
		}
		if options.Io.Err != nil {
			cmd.Stderr = io.MultiWriter(options.Io.Err, &stderrBuf)
		} else {
			cmd.Stderr = io.MultiWriter(combined, &stderrBuf)
		}
		err := cmd.Run()
		// Note: while it is tempting to trim the trailing \n here, some code flows require it,
		//       namely `git diff | git apply`.`
		stringOut := combined.String()
		if err != nil && retry != nil && retry(executionCount, stderrBuf.String()) {
			fullCommand := programName + " " + strings.Join(flatArgs, " ")
			firstLineCmd, _, _ := strings.Cut(fullCommand, "\n")
			firstLineError, _, _ := strings.Cut(stderrBuf.String(), "\n")
			slog.Warn("Retrying: " + "\"" + firstLineCmd + "\": " + firstLineError)
			continue
		}
		slog.Debug("Executed " + getLogMessage(programName, flatArgs, stringOut, err))
		return stringOut, err
	}
}

// Executes a shell program with arguments.
func Execute(options ExecuteOptions, programName string, args ...any) (string, error) {
	return globalExecutor.Execute(options, programName, args...)
}

// Executes a shell program with arguments. Panics if there is an error.
func ExecuteOrDie(options ExecuteOptions, programName string, args ...any) string {
	out, err := Execute(options, programName, args...)
	if err != nil {
		flatArgs := flattenArgs(args)
		panic("failed executing " + getLogMessage(programName, flatArgs, out, err))
	}
	return out
}

// Executes a shell program with arguments, trims whitespace from output, and panics if there is an error.
func ExecuteOrDieTrimmed(options ExecuteOptions, programName string, args ...any) string {
	return strings.TrimSpace(ExecuteOrDie(options, programName, args...))
}

func getLogMessage(programName string, args []string, out string, err error) string {
	var logMessage string
	if err != nil {
		logMessage = logMessage + "(" + err.Error() + ") "
	}
	logMessage += "\"" + programName + " " + strings.Join(args, " ") + "\""
	if strings.TrimSpace(out) != "" {
		logMessage = logMessage + "\n\n" + strings.TrimSuffix(out, "\n")
	}
	return logMessage
}
