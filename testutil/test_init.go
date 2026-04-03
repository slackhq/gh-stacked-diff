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
	"testing"
	"time"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

const InitialCommitSubject = "Initial empty commit"
const programName string = "gh-stacked-diff"

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
	util.SetUserConfig(util.UserConfig{})

	cdTestRepo(testFunctionName)
	// Setup author config in case it is not set on machine.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "config", "user.email", "unit-test@example.com")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "config", "user.name", "Unit Test")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "--allow-empty", "-m", InitialCommitSubject)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetCurrentBranchName())
	// Set origin/HEAD so GetRemoteMainBranch works.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "remote", "set-head", "origin", util.GetCurrentBranchName())

	util.SetDefaultSleep(func(d time.Duration) {
		slog.Debug(fmt.Sprint("Skipping sleep in tests ", d))
	})

	panicOnExit := func(code int) {
		panic("Panicking instead of exiting with code " + fmt.Sprint(code))
	}
	// Executable must be on PATH for tests to pass so that sequenceEditorPrefix will execute.
	// PATH is set in ../Makefile
	appExecutable := programName

	// Set stdin in unit tests to avoid error with bubbletea:
	// "error creating cancelreader: failed to prepare console input: get console mode: The handle is invalid."
	// To fake user input use interactive.SendToProgram.
	stdin := strings.NewReader("")

	appConfig := util.AppConfig{
		Io:            util.StdIo{Out: os.Stdout, Err: os.Stdout, In: stdin},
		AppExecutable: appExecutable,
		Exit:          panicOnExit,
		UserCacheDir:  getTestAppCacheDir(),
		ConfigHome:    getTestConfigHome(t),
		DemoMode:      false,
	}
	util.SetAppConfig(appConfig)

	util.RunTestInitHooks(t)
	return testExecutor
}

func getTestConfigHome(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		panic("cannot get wd: " + err.Error())
	}
	parentDir, _ := filepath.Split(wd)
	configHome := filepath.Join(parentDir, ".gh-stacked-diff")
	// nolint:errcheck
	os.Mkdir(configHome, os.ModePerm)
	configFile := filepath.Join(configHome, "config.yaml")
	// nolint:errcheck
	os.Remove(configFile)
	t.Cleanup(func() {
		// nolint:errcheck
		os.Remove(configFile)
	})
	return configHome
}

func getTestAppCacheDir() string {
	// okay I need it as a C:\\ in order to use WriteFile/ReadFile
	// but all of the path stuff uses /
	wd, err := os.Getwd()
	if err != nil {
		panic("cannot get wd: " + err.Error())
	}
	parentDir, _ := filepath.Split(wd)
	userCacheDir := filepath.Join(parentDir, "user-cache")
	// nolint:errcheck
	os.Mkdir(userCacheDir, os.ModePerm)
	return userCacheDir
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

// WaitForOutput polls the stringer until its output contains the expected
// substring, or fails the test after 10 seconds.
func WaitForOutput(t *testing.T, out fmt.Stringer, expected string) {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for !strings.Contains(out.String(), expected) {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for output to contain %q", expected)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// WaitForDone waits for the channel to close, or fails the test after 10 seconds.
func WaitForDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for done")
	}
}

// SetupSecondaryWorktree creates a secondary worktree from the current repo
// and changes into it. Resets cached branch values.
// Returns the main worktree path.
func SetupSecondaryWorktree(t *testing.T) string {
	t.Helper()
	mainPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "secondary-branch")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../secondary-worktree", "secondary-branch")
	t.Cleanup(func() {
		_ = os.Chdir(mainPath)
		_, _ = util.Execute(util.ExecuteOptions{}, "git", "worktree", "remove", "--force", "../secondary-worktree")
	})
	if err := os.Chdir("../secondary-worktree"); err != nil {
		t.Fatal(err)
	}
	return mainPath
}

func CommitFileChange(commitMessage string, filename string, fileContents string) {
	util.ExecuteOrDie(util.ExecuteOptions{}, "touch", filename)
	if writeErr := os.WriteFile(filename, []byte(fileContents), 0); writeErr != nil {
		panic(writeErr)
	}
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "-m", commitMessage)
}
