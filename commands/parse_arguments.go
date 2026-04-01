package commands

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"strings"

	"github.com/fatih/color"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func ExecuteCommand(appConfig util.AppConfig, commandLineArgs []string) {
	util.SetAppConfig(appConfig)

	// Unset any color in case a previous terminal command set colors and then was
	// terminated before it could reset the colors.
	color.Unset()

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

	// Set up logger early so debug output is visible during cobra parsing.
	setSlogLogger(appConfig.Io.Out, findLogLevelArg(commandLineArgs))

	slog.Debug("App executable: " + appConfig.AppExecutable)
	slog.Debug("User cache dir: " + appConfig.UserCacheDir)

	rootCmd := buildRootCommand()
	rootCmd.SetArgs(commandLineArgs)

	if err := rootCmd.Execute(); err != nil {
		panic(err.Error())
	}
}

func buildRootCommand() *cobra.Command {
	appConfig := util.GetAppConfig()
	var stableSuffix string
	if strings.TrimSpace(util.CurrentVersion) == strings.TrimSpace(util.StableVersion) {
		stableSuffix = " (stable)"
	} else {
		stableSuffix = " (preview)"
	}
	rootCmd := &cobra.Command{
		Use:           "sd [flags] <command> [args]",
		Short:         "Stacked Diff Workflow",
		Version:       strings.TrimSpace(util.CurrentVersion) + stableSuffix,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.SetOut(appConfig.Io.Out)
	rootCmd.SetErr(appConfig.Io.Err)

	var logLevelString string
	rootCmd.PersistentFlags().StringVarP(&logLevelString, "log-level", "l", "",
		"Possible log levels:\n"+
			"   debug\n"+
			"   info\n"+
			"   warn\n"+
			"   error\n"+
			"Default is info, except on commands that are for output purposes,\n"+
			"(namely branch-name and log), which have a default of error.")

	_ = rootCmd.RegisterFlagCompletionFunc("log-level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"debug", "info", "warn", "error"}, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.PersistentFlags().StringToStringP("config", "c", nil,
		"Set a config value as key=value. Overrides values from\n"+
			"~/.gh-stacked-diff/config.yaml. Supported keys:\n"+
			"   promptForReview=never|promptY|promptN (default: promptN)\n"+
			"   pollInterval=<duration> (default: 30s, e.g. 1m, 10s)\n"+
			"   ticketUrlPattern=<url> URL pattern for tickets, e.g.\n"+
			"                          https://jira.example.com/browse/{TicketNumber}\n"+
			"Can be specified multiple times for different keys.\n"+
			"\n"+
			"Equivalent config.yaml:\n"+
			"   promptForReview: promptY\n"+
			"   pollInterval: 1m\n"+
			"   ticketUrlPattern: https://jira.example.com/browse/{TicketNumber}")
	rootCmd.PersistentFlags().Lookup("config").DefValue = ""

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// If --log-level flag was set, it was already applied in ExecuteCommand.
		// Otherwise, use command annotation default, falling back to info.
		if logLevelString == "" {
			defaultLevel := slog.LevelInfo
			if ann, ok := cmd.Annotations["defaultLogLevel"]; ok {
				defaultLevel = stringToLogLevel(ann)
			}
			setSlogLogger(appConfig.Io.Out, defaultLevel)
		}
		if cmd.Annotations["skipRepoCheck"] != "true" {
			// Note: call GetMainBranchOrDie early as it has useful error messages.
			slog.Debug("Using main branch " + util.GetMainBranchOrDie())
		}
	}

	rootCmd.AddCommand(
		createAddDescriptionCommand(),
		createAddReviewersCommand(),
		createBranchNameCommand(),
		createCheckoutCommand(),
		createCodeOwnersCommand(),
		createDropAlreadyMergedCommand(),
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
	)

	return rootCmd
}

func setSlogLogger(stdOut io.Writer, logLevel slog.Level) {
	handler := util.NewPrettyHandler(stdOut, slog.HandlerOptions{Level: logLevel})
	slog.SetDefault(slog.New(handler))
}

// findLogLevelArg scans command line args for --log-level or -l and returns
// the parsed level. This allows setting up the logger before cobra parses.
func findLogLevelArg(args []string) slog.Level {
	for i, arg := range args {
		var value string
		if strings.HasPrefix(arg, "--log-level=") {
			value = strings.TrimPrefix(arg, "--log-level=")
		} else if strings.HasPrefix(arg, "-l=") {
			value = strings.TrimPrefix(arg, "-l=")
		} else if (arg == "--log-level" || arg == "-l") && i+1 < len(args) {
			value = args[i+1]
		} else {
			continue
		}
		return stringToLogLevel(value)
	}
	return slog.LevelInfo
}

func stringToLogLevel(logLevelString string) slog.Level {
	if logLevelString == "" {
		return slog.LevelInfo
	}
	var logLevel slog.Level
	var unmarshallErr = logLevel.UnmarshalText([]byte(logLevelString))
	if unmarshallErr != nil {
		panic("invalid log level: \"" + logLevelString + "\"")
	}
	return logLevel
}
