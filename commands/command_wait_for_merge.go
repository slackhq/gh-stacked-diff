package commands

import (
	"log/slog"
	"time"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createWaitForMergeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-for-merge [commitIndicator]",
		Short: "Waits for a pull request to be merged",
		Long: "Waits for a pull request to be merged. Poll interval is configurable via --config pollInterval.\n" +
			"\n" +
			"Useful for your own custom scripting.",
		Args: cobra.MaximumNArgs(1),
	}
	indicatorTypeString := addIndicatorFlag(cmd)
	silent := addSilentFlag(cmd, "the PR has been merged")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		userConfig := util.GetUserConfig()
		selectCommitOptions := interactive.CommitSelectionOptions{
			Prompt:      "What PR do you want to wait for to be merged?",
			CommitType:  interactive.CommitTypePr,
			MultiSelect: false,
		}
		targetCommit := getTargetCommits(args, indicatorTypeString, selectCommitOptions)
		waitForMerge(targetCommit[0], *silent, userConfig.PollInterval)
	}
	return cmd
}

// Waits for a pull request to be merged.
func waitForMerge(targetCommit templates.GitLog, silent bool, pollInterval time.Duration) {
	for getMergedAt(targetCommit.Branch) == "" {
		slog.Info("Not merged yet...")
		util.Sleep(pollInterval)
	}
	slog.Info("Merged!")
	if !silent {
		util.ExecuteOrDie(util.ExecuteOptions{}, "say", "P R has been merged")
	}
}

func getMergedAt(branchName string) string {
	return util.ExecuteOrDieTrimmed(util.ExecuteOptions{Retries: gitutil.GhRetries},
		"gh", "pr", "view", branchName, "--json", "mergedAt", "--jq", ".mergedAt")
}
