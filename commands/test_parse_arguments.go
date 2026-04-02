package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

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
	appConfig := util.GetAppConfig()
	appConfig.Io = util.StdIo{Out: out, Err: out, In: appConfig.Io.In}

	if !slices.ContainsFunc(commandLineArgs, func(next string) bool {
		return strings.HasPrefix(next, "--log-level")
	}) {
		// Use current log level if it set to something other than Info.
		level := lowestSupportedLogLevel()
		if level != slog.LevelInfo {
			loglevelFlag := "--log-level=" + level.String()
			appConfig.AppExecutable += " " + loglevelFlag
			commandLineArgs = slices.Insert(commandLineArgs, 0, loglevelFlag)
		}
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
