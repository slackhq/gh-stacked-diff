package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func ExecuteCommand(appConfig util.AppConfig, commandLineArgs []string) {
	// Unset any color in case a previous terminal command set colors and then was
	// terminated before it could reset the colors.
	color.Unset()

	parseArguments(appConfig, flag.NewFlagSet("sd", flag.ContinueOnError), commandLineArgs)
}

func parseArguments(appConfig util.AppConfig, commandLine *flag.FlagSet, commandLineArgs []string) {
	defer func() {
		r := recover()
		if r != nil {
			util.Fprintln(appConfig.Io.Err, color.RedString(fmt.Sprint("error: ", r)))
			if slog.Default().Handler().Enabled(context.Background(), slog.LevelDebug) {
				util.Fprintln(appConfig.Io.Err, string(debug.Stack()))
			}
			appConfig.Exit(1)
		}
	}()
	if commandLine.ErrorHandling() != flag.ContinueOnError {
		// Use ContinueOnError so that a description of the command can be included before usage
		// for help.
		panic("ErrorHandling must be ContinueOnError, not " + fmt.Sprint(commandLine.ErrorHandling()))
	}
	// clear FlagSet.Usage and discard any output so that it is not display automatically on an invalid parameter.
	commandLine.Usage = func() {}
	commandLine.SetOutput(io.Discard)
	// Parse top level flags.
	logLevelString := commandLine.String("log-level", "",
		"Possible log levels:\n"+
			"   debug\n"+
			"   info\n"+
			"   warn\n"+
			"   error\n"+
			"Default is info, except on commands that are for output purposes,\n"+
			"(namely branch-name and log), which have a default of error.")
	parseErr := commandLine.Parse(commandLineArgs)
	var logLevelVar *slog.LevelVar
	if parseErr == nil {
		// allow for setting of log level to DEBUG so that the very first execute statements can be logged.
		// logLevel will be potentially set again once we know what command is executed.
		var logLevel slog.Level
		logLevel, parseErr = stringToLogLevel(*logLevelString)
		if parseErr == nil {
			logLevelVar = setSlogLogger(appConfig.Io.Out, logLevel)
		}
	}
	// parseErr is dealt with below via commandError and commandHelp.

	commands := []Command{
		createAddDescriptionCommand(),
		createAddReviewersCommand(),
		createBranchNameCommand(),
		createCheckoutCommand(),
		createCodeOwnersCommand(),
		createDropAlreadyMergedCommand(),
		createCompletionCommand(),
		createLogCommand(),
		createMarkAsFixupCommand(),
		createMigrateCommand(),
		createNewCommand(),
		createPrsCommand(),
		createRebaseMainCommand(),
		createReplaceCommitCommand(),
		createReplaceConflictsCommand(),
		createUpdateCommand(),
		createVersionCommand(),
		createWaitForMergeCommand(),
	}

	commandLineDescription := "Stacked Diff Workflow"
	commandLineUsage := "sd [top-level-flags] <command> [<args>]\n" +
		"\n" +
		"Possible commands are:\n" +
		"\n" +
		"   " + strings.Join(getCommandSummaries(commands), "\n   ") + "\n" +
		"\n" +
		"To learn more about a command use: sd <command> --help"

	if parseErr != nil {
		if parseErr == flag.ErrHelp {
			commandHelp(appConfig, commandLine, commandLineDescription, commandLineUsage, false)
		} else {
			commandError(appConfig, commandLine, parseErr.Error(), commandLineUsage)
		}
	}

	if commandLine.NArg() == 0 {
		commandHelp(appConfig, commandLine, commandLineDescription, commandLineUsage, true)
	}
	selectedIndex := slices.IndexFunc(commands, func(command Command) bool {
		return command.FlagSet.Name() == commandLine.Arg(0)
	})
	if selectedIndex == -1 {
		commandError(appConfig, commandLine, "unknown command "+commandLine.Arg(0), commandLineUsage)
	}

	if commands[selectedIndex].FlagSet.ErrorHandling() != flag.ContinueOnError {
		panic("ErrorHandling must be ContinueOnError, not " + fmt.Sprint(commands[selectedIndex].FlagSet.ErrorHandling()))
	}
	commands[selectedIndex].FlagSet.Usage = func() {}
	commands[selectedIndex].FlagSet.SetOutput(io.Discard)
	if parseErr := commands[selectedIndex].FlagSet.Parse(commandLine.Args()[1:]); parseErr != nil {
		if parseErr == flag.ErrHelp {
			commandHelp(appConfig, commands[selectedIndex].FlagSet, commands[selectedIndex].Description, commands[selectedIndex].Usage, false)
		} else {
			commandError(appConfig, commands[selectedIndex].FlagSet, parseErr.Error(), commands[selectedIndex].Usage)
		}
	}

	if *logLevelString == "" {
		logLevelVar.Set(commands[selectedIndex].DefaultLogLevel)
	}
	slog.Debug("App executable: " + appConfig.AppExecutable)
	slog.Debug("User cache dir: " + appConfig.UserCacheDir)
	if !commands[selectedIndex].SkipRepoCheck {
		// Note: call GetMainBranchOrDie early as it has useful error messages.
		slog.Debug(fmt.Sprint("Using main branch " + util.GetMainBranchOrDie()))
	}
	commands[selectedIndex].OnSelected(appConfig, commands[selectedIndex])
}

func getCommandSummaries(commands []Command) []string {
	publicCommands := util.FilterSlice(commands, func(command Command) bool {
		return !command.Hidden
	})
	maxCommandLen := 0
	for _, command := range publicCommands {
		if len(command.FlagSet.Name()) > maxCommandLen {
			maxCommandLen = len(command.FlagSet.Name())
		}
	}
	summaries := make([]string, 0, len(commands))
	for _, command := range publicCommands {
		summary := command.FlagSet.Name() + "   " + strings.Repeat(" ", maxCommandLen-len(command.FlagSet.Name())) + command.Summary
		summaries = append(summaries, summary)
	}
	slices.Sort(summaries)
	return summaries
}

func setSlogLogger(stdOut io.Writer, logLevel slog.Level) *slog.LevelVar {
	var levelVar slog.LevelVar
	levelVar.Set(logLevel)
	handler := util.NewPrettyHandler(stdOut, slog.HandlerOptions{Level: &levelVar})
	slog.SetDefault(slog.New(handler))
	return &levelVar
}

func stringToLogLevel(logLevelString string) (slog.Level, error) {
	if logLevelString == "" {
		return slog.LevelInfo, nil
	}
	var logLevel slog.Level
	var unmarshallErr = logLevel.UnmarshalText([]byte(logLevelString))
	if unmarshallErr != nil {
		return 0, unmarshallErr
	}
	return logLevel, nil
}
