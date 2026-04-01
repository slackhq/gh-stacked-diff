package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// Program name
const programName string = "gh-stacked-diff"

// Calls [ExecuteCommand] for unit tests.
func testParseArguments(commandLineArgs ...string) string {
	if slog.Default().Handler().Enabled(context.Background(), slog.LevelInfo) {
		out := util.NewWriteRecorder(os.Stdout)
		testParseArgumentsWithOut(out, commandLineArgs...)
		return out.String()
	} else {
		out := new(bytes.Buffer)
		testParseArgumentsWithOut(out, commandLineArgs...)
		return out.String()
	}
}

func testParseArgumentsWithOut(out io.Writer, commandLineArgs ...string) {
	slog.Debug(fmt.Sprint("***Testing parse arguments*** ", strings.Join(commandLineArgs, " ")))
	panicOnExit := func(code int) {
		panic("Panicking instead of exiting with code " + fmt.Sprint(code))
	}
	// Executable must be on PATH for tests to pass so that sequenceEditorPrefix will execute.
	// PATH is set in ../Makefile
	appExecutable := programName

	if !slices.ContainsFunc(commandLineArgs, func(next string) bool {
		return strings.HasPrefix(next, "--log-level")
	}) {
		// Use current log level if it set to something other than Info.
		level := lowestSupportedLogLevel()
		if level != slog.LevelInfo {
			loglevelFlag := "--log-level=" + level.String()
			appExecutable += " " + loglevelFlag
			commandLineArgs = slices.Insert(commandLineArgs, 0, loglevelFlag)
		}
	}

	// Set stdin in unit tests to avoid error with bubbletea:
	// "error creating cancelreader: failed to prepare console input: get console mode: The handle is invalid."
	// To fake user input use interactive.SendToProgram.
	stdin := strings.NewReader("")

	appConfig := util.AppConfig{
		Io:            util.StdIo{Out: out, Err: out, In: stdin},
		AppExecutable: appExecutable,
		Exit:          panicOnExit,
		UserCacheDir:  getTestAppCacheDir(),
		ConfigHome:    getTestConfigHome(),
		DemoMode:      false,
	}
	ExecuteCommand(appConfig, commandLineArgs)
	slog.Debug(fmt.Sprint("***Done running arguments*** ", strings.Join(commandLineArgs, " ")))
}

func lowestSupportedLogLevel() slog.Level {
	levels := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn}
	for _, level := range levels {
		if slog.Default().Handler().Enabled(context.Background(), level) {
			return level
		}
	}
	return slog.LevelError
}

func getTestConfigHome() string {
	wd, err := os.Getwd()
	if err != nil {
		panic("cannot get wd: " + err.Error())
	}
	parentDir, _ := filepath.Split(wd)
	configHome := filepath.Join(parentDir, ".gh-stacked-diff")
	// nolint:errcheck
	os.Mkdir(configHome, os.ModePerm)
	// Write a default config.yaml so that tests don't trigger interactive
	// prompts for ticketUrlPattern.
	configFile := filepath.Join(configHome, "config.yaml")
	if _, statErr := os.Stat(configFile); os.IsNotExist(statErr) {
		// nolint:errcheck
		os.WriteFile(configFile, []byte("ticketUrlPattern: https://example.com/browse/{TicketNumber}\n"), 0644)
	}
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
