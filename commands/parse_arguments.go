package commands

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"strings"

	"github.com/fatih/color"
	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

const checkRepoAnnotation = "checkRepo"

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
	slog.Debug("User cache dir: " + appConfig.CacheDir())

	rootCmd := buildRootCommand()
	rootCmd.SetArgs(commandLineArgs)

	if err := rootCmd.Execute(); err != nil {
		panic(err.Error())
	}
}

func buildRootCommand() *cobra.Command {
	appConfig := util.GetAppConfig()
	rootCmd := &cobra.Command{
		Use:           "sd [flags] <command> [args]",
		Short:         "Stacked Diff Workflow",
		Version:       strings.TrimSpace(util.CurrentVersion) + util.VersionSuffix(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.SetOut(appConfig.Io.Out)
	rootCmd.SetErr(appConfig.Io.Err)
	cobra.AddTemplateFunc("rpadx", func(s string, padding int) string {
		return fmt.Sprintf("%-*s", padding, s)
	})
	rootCmd.SetUsageTemplate(strings.Replace(rootCmd.UsageTemplate(), "rpad .Name .NamePadding", "rpadx .Name 20", 1))

	var logLevelString string
	rootCmd.PersistentFlags().StringVarP(&logLevelString, "log-level", "l", "",
		"Set log level: debug|info|warn|error (default: info)")

	_ = rootCmd.RegisterFlagCompletionFunc("log-level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"debug", "info", "warn", "error"}, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.PersistentFlags().StringToStringP("config", "c", nil,
		"Set a config key=value, overrides config.yaml (see 'sd flags' for keys)")
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
		configValues, err := cmd.Flags().GetStringToString("config")
		if err != nil {
			panic(err.Error())
		}
		fileConfig := util.LoadUserConfigFile()
		metrics := util.LoadMetricsFile()
		util.SetUserConfig(util.NewUserConfig(fileConfig, configValues, metrics))
		if cmd.Annotations[checkRepoAnnotation] == "true" {
			if _, err := util.Execute(util.ExecuteOptions{}, "git", "rev-parse", "--is-inside-work-tree"); err != nil {
				panic("Not in a git repository. Must be run from a git repository")
			}
			// Note: call GetLocalMainBranchOrDie early as it has useful error messages.
			slog.Debug("Using main branch " + gitutil.GetLocalMainBranchOrDie())
		}
	}

	rootCmd.AddCommand(
		createAddDescriptionCommand(),
		createAddReviewersCommand(),
		createBranchNameCommand(),
		createCheckoutCommand(),
		createCodeOwnersCommand(),
		createDropAlreadyMergedCommand(),
		createFlagsCommand(),
		createLogCommand(),
		createMarkAsFixupCommand(),
		createMigrateCommand(),
		createNewCommand(),
		createPrsCommand(),
		createRebaseMainCommand(),
		createReplaceCommitCommand(),
		createReplaceConflictsCommand(),
		createUpdateBranchesCommand(),
		createUpdateCommand(),
		createVersionCommand(),
		createWaitForMergeCommand(),
		createWorktreeMoveCommand(),
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
