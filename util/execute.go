package util

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

const RetryDelay = 1 * time.Second

// Options for [ExecuteWithOptions].
type ExecuteOptions struct {
	// What to use for input and output. Overriding input is useful for "git apply"
	// If output is not set then output is returned from Execute.
	// Any nil In/Err/Out values are ignored.
	Io StdIo
	// For example "MY_VAR=some_value"
	EnvironmentVariables []string
	Retries              int
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
	cmd := exec.Command(programName, flatArgs...)
	if options.EnvironmentVariables != nil {
		cmd.Env = append(os.Environ(), options.EnvironmentVariables...)
	}
	if options.Io.In != nil {
		cmd.Stdin = options.Io.In
	}
	var b bytes.Buffer
	if options.Io.Out != nil {
		cmd.Stdout = options.Io.Out
	} else {
		cmd.Stdout = &b
	}
	if options.Io.Err != nil {
		cmd.Stderr = options.Io.Err
	} else {
		cmd.Stderr = &b
	}
	err := cmd.Run()
	// Note: while it is tempting to trim the trailing \n here, some code flows require it,
	//       namely `git diff | git apply`.`
	stringOut := b.String()
	if err != nil && options.Retries > 0 {
		fullCommand := programName + " " + strings.Join(flatArgs, " ")
		firstLine, _, _ := strings.Cut(fullCommand, "\n")
		slog.Warn("Retrying: " + "\"" + firstLine + "\": " + err.Error())
		Sleep(RetryDelay)
		options.Retries = options.Retries - 1
		return defaultExecutor.Execute(options, programName, args...)
	}
	slog.Debug("Executed " + getLogMessage(programName, flatArgs, stringOut, err))
	return stringOut, err
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
