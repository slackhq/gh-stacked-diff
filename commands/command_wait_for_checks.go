package commands

import (
	"slices"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createWaitForChecksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-for-checks [commitIndicator...]",
		Short: "Wait for the checks of a Pull Request on Github to pass",
		Long: "Wait for the checks of a Pull Request on Github to pass.\n" +
			"\n" +
			"Like add-reviewers, but does not mark the PR as ready for review or add any reviewers.",
		Annotations: map[string]string{
			checkRepoAnnotation: "true",
		},
	}
	indicatorTypeString := addIndicatorFlag(cmd)
	silent := addSilentFlag(cmd, "checks have failed")
	minChecks := cmd.Flags().Int("min-checks", -1,
		"Minimum number of checks to wait for before verifying that checks\n"+
			"have passed. It takes some time for checks to be added to a PR by\n"+
			"Github, and if you wait-for-checks too soon it will think that they\n"+
			"have all passed. Default of -1 means to use 4 or the average number\n"+
			"of checks of merged PRs, whatever is less.")

	cmd.Run = func(cmd *cobra.Command, args []string) {
		userConfig := util.GetUserConfig()
		selectPrsOptions := interactive.CommitSelectionOptions{
			Prompt:      "What PR do you want to wait for checks on?",
			CommitType:  interactive.CommitTypePr,
			MultiSelect: true,
		}
		targetCommits := getTargetCommits(args, indicatorTypeString, selectPrsOptions)
		// Reverse the order as getTargetCommits returns cherry-pick order and we want to display in log order.
		slices.Reverse(targetCommits)
		addReviewersToPr(targetCommits, AddReviewersOptions{
			WhenChecksPass:    true,
			Silent:            *silent,
			MinChecks:         *minChecks,
			PollFrequency:     userConfig.PollInterval,
			OnlyWaitForChecks: true,
		})
	}
	return cmd
}
