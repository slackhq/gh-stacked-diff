package commands

import (
	"log/slog"
	"runtime"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func addIndicatorFlag(cmd *cobra.Command) *string {
	usage := "Indicator type to use to interpret commitIndicator:\n" +
		"   commit   a commit hash, can be abbreviated,\n" +
		"   pr       a github Pull Request number,\n" +
		"   list     the order of commit listed in the git log, as indicated\n" +
		"            by \"sd log\"\n" +
		"   guess    the command will guess the indicator type:\n" +
		"      Number between 0 and 99:       list\n" +
		"      Number between 100 and 999999: pr\n" +
		"      Otherwise:                     commit\n"
	indicator := cmd.Flags().StringP("indicator", "i", string(templates.IndicatorTypeGuess), usage)
	_ = cmd.RegisterFlagCompletionFunc("indicator", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"commit", "pr", "list", "guess"}, cobra.ShellCompDirectiveDefault
	})
	return indicator
}

func checkIndicatorFlag(indicatorTypeString *string) templates.IndicatorType {
	indicatorType := templates.IndicatorType(*indicatorTypeString)
	if !indicatorType.IsValid() {
		panic("Invalid indicator type: " + *indicatorTypeString)
	}
	return indicatorType
}

func addReviewersFlags(cmd *cobra.Command, appConfig util.AppConfig) (*string, *bool, *int, *bool) {
	reviewers := cmd.Flags().StringP("reviewers", "r", "",
		"Comma-separated list of Github usernames to add as reviewers once\n"+
			"checks have passed.")
	_ = cmd.RegisterFlagCompletionFunc("reviewers", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return interactive.ReviewersHistory.ReadHistory(appConfig), cobra.ShellCompDirectiveNoFileComp
	})
	silent := addSilentFlag(cmd, "reviewers have been added")
	minChecks := cmd.Flags().Int("min-checks", -1,
		"Minimum number of checks to wait for before verifying that checks\n"+
			"have passed before adding reviewers. It takes some time for checks\n"+
			"to be added to a PR by Github, and if you add-reviewers too soon it\n"+
			"will think that they have all passed. Default of -1 means to use 4 \n"+
			"or the average number of checks of merged PRs, whatever is less.")
	merge := cmd.Flags().BoolP("merge", "m", false, "Enable auto-merge (squash) on the PR via Github CLI.")
	return reviewers, silent, minChecks, merge
}

func addSilentFlag(cmd *cobra.Command, usageUseCase string) *bool {
	if runtime.GOOS == "darwin" {
		// Only supported on Mac OS X because it uses "say" command.
		return cmd.Flags().BoolP("silent", "s", false, "Whether to use voice output (false) or be silent (true) to notify that "+usageUseCase+".")
	} else {
		silent := new(bool)
		*silent = true
		return silent
	}
}

// promptForReviewers handles the common pattern of optionally prompting the user
// to mark a PR as ready for review and select reviewers. Returns whether the user
// chose to mark the PR as ready.
func promptForReviewers(appConfig util.AppConfig, reviewers *string, shouldPrompt bool, userConfig UserConfig) bool {
	if *reviewers != "" || !shouldPrompt {
		return false
	}
	var markReady bool
	switch userConfig.PromptForReview() {
	case util.PromptForReviewNever:
		return false
	case util.PromptForReviewPromptY:
		markReady = interactive.Confirm(appConfig, "Mark PR as ready for review when checks pass?", true)
	case util.PromptForReviewPromptN:
		markReady = interactive.Confirm(appConfig, "Mark PR as ready for review when checks pass?", false)
	}
	if markReady {
		*reviewers = interactive.UserSelection(appConfig, true)
		if *reviewers != "" {
			slog.Info("Using reviewers " + *reviewers)
		}
	}
	return markReady
}

// sequenceEditorEnvVar builds the GIT_SEQUENCE_EDITOR environment variable string
// that invokes the app executable with the given subcommand and arguments.
func sequenceEditorEnvVar(appConfig util.AppConfig, subcommand string, args ...string) string {
	parts := []string{appConfig.AppExecutable, subcommand}
	parts = append(parts, args...)
	return "GIT_SEQUENCE_EDITOR=" + strings.Join(parts, " ")
}
