/*
Utilities for unit testing this project.
*/
package testutil

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

const InitialCommitSubject = "Initial empty commit"

var TestWorkingDir string
var thisFile string

func init() {
	_, file, _, ok := runtime.Caller(0)
	thisFile = file
	if !ok {
		panic("No caller information")
	}
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic("Cannot find UserCacheDir: " + err.Error())
	}
	TestWorkingDir = filepath.Join(userCacheDir, "gh-stacked-diff-tests")
	// nolint:errcheck
	os.Mkdir(TestWorkingDir, os.ModePerm)
}

// CD into repository directory and set any global DI variables (slog, sleep, and executor).
func InitTest(t *testing.T, logLevel slog.Level) *util.TestExecutor {
	handler := util.NewPrettyHandler(os.Stdout, slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(handler))
	testFunctionName := getTestFunctionName()

	// Set new TestExecutor in case previous test has faked any of the git responses.
	testExecutor := setTestExecutor()

	cdTestRepo(testFunctionName)
	// Setup author config in case it is not set on machine.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "config", "user.email", "unit-test@example.com")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "config", "user.name", "Unit Test")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "--allow-empty", "-m", InitialCommitSubject)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetCurrentBranchName())

	util.SetDefaultSleep(func(d time.Duration) {
		slog.Debug(fmt.Sprint("Skipping sleep in tests ", d))
	})

	interactive.RequireInput(t)
	return testExecutor
}

func getTestFunctionName() string {
	var functionName string
	for i := 0; i < 10; i++ {
		pc, file, _, ok := runtime.Caller(i)
		if !ok {
			panic("No caller information")
		}
		if file != thisFile {
			functionName = runtime.FuncForPC(pc).Name()
			break
		}
	}
	if functionName == "" {
		panic("Could not find caller outside of " + thisFile)
	}
	// Reduce the length of the function name as otherwise on windows the OS max can be exceeded.
	functionParts := strings.Split(functionName, "/")
	return functionParts[len(functionParts)-1]
}

func cdTestRepo(testFunctionName string) {
	cdTestDir(testFunctionName)
	// Create a git repository with a local remote
	remoteDir := "remote-repo"
	repositoryDir := "local-repo"

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "init", "--bare", remoteDir)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "clone", remoteDir, repositoryDir)

	if err := os.Chdir(repositoryDir); err != nil {
		panic(err)
	}
	// os.Getwd() returns an unusable path ("c:\..."") in windows when running from Git Bash. Instead log "pwd"
	wd := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "pwd")
	slog.Info("Changed to test repository directory:\n" + wd)
}

func cdTestDir(testFunctionName string) {
	individualTestDir := filepath.Join(TestWorkingDir, testFunctionName)
	util.ExecuteOrDie(util.ExecuteOptions{}, "rm", "-rf", individualTestDir)
	// Note: Using "mkdir -p" hangs sometimes on Windows, so use os.Mkdir instead.
	// nolint:errcheck
	os.Mkdir(individualTestDir, os.ModePerm)
	if err := os.Chdir(individualTestDir); err != nil {
		panic(err)
	}
}

func setTestExecutor() *util.TestExecutor {
	testExecutor := util.TestExecutor{}
	testExecutor.SetResponse("Ok", nil, "gh", util.MatchAnyRemainingArgs)
	testExecutor.SetResponse("Ok", nil, "say", util.MatchAnyRemainingArgs)
	util.SetGlobalExecutor(&testExecutor)
	return &testExecutor
}

func AddCommit(commitMessage string, filename string) {
	if filename == "" {
		filename = commitMessage
	}
	util.ExecuteOrDie(util.ExecuteOptions{}, "touch", filename)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "-m", commitMessage)
}

func CommitFileChange(commitMessage string, filename string, fileContents string) {
	util.ExecuteOrDie(util.ExecuteOptions{}, "touch", filename)
	if writeErr := os.WriteFile(filename, []byte(fileContents), 0); writeErr != nil {
		panic(writeErr)
	}
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "-m", commitMessage)
}
