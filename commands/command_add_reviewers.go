package commands

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

const waitForOtherReviewersSeconds = 10

func createAddReviewersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-reviewers [commitIndicator...]",
		Short: "Add reviewers to Pull Request on Github once its checks have passed",
		Long: "Add reviewers to Pull Request on Github once its checks have passed.\n" +
			"\n" +
			"If PR is marked as a Draft, it is first marked as \"Ready for Review\".",
		Annotations: map[string]string{
			checkRepoAnnotation: "true",
		},
	}
	indicatorTypeString := addIndicatorFlag(cmd)

	whenChecksPass := cmd.Flags().BoolP("when-checks-pass", "w", true, "Poll until all checks pass before adding reviewers")
	reviewers, silent, minChecks, merge := addReviewersFlags(cmd)

	cmd.Run = func(cmd *cobra.Command, args []string) {
		userConfig := util.GetUserConfig()
		selectPrsOptions := interactive.CommitSelectionOptions{
			Prompt:      "What PR do you want to add reviewers to?",
			CommitType:  interactive.CommitTypePr,
			MultiSelect: true,
		}
		targetCommits := getTargetCommits(args, indicatorTypeString, selectPrsOptions)
		// Reverse the order as getTargetCommits returns cherry-pick order and we want to display in log order.
		slices.Reverse(targetCommits)
		if *reviewers == "" {
			*reviewers = interactive.UserSelection()
		}
		if *reviewers != "" {
			slog.Info("Using reviewers " + *reviewers)
		}
		addReviewersToPr(targetCommits, AddReviewersOptions{
			WhenChecksPass: *whenChecksPass,
			Silent:         *silent,
			MinChecks:      *minChecks,
			Reviewers:      *reviewers,
			PollFrequency:  userConfig.PollInterval,
			AutoMerge:      *merge,
		})
	}
	return cmd
}

type AddReviewersOptions struct {
	WhenChecksPass    bool
	Silent            bool
	MinChecks         int
	Reviewers         string
	PollFrequency     time.Duration
	AutoMerge         bool
	WaitBeforePolling time.Duration
}

// Adds reviewers to a PR once checks have passed via Github CLI.
func addReviewersToPr(targetCommits []templates.GitLog, opts AddReviewersOptions) {
	if opts.Reviewers != "" {
		interactive.ReviewersHistory.AddToHistory(opts.Reviewers)
	}
	progressIndicatorMessages := util.MapSlice(targetCommits, func(next templates.GitLog) string {
		return next.String()
	})
	progressIndicator := interactive.NewProgressIndicator(progressIndicatorMessages)
	var wg sync.WaitGroup
	for i, targetCommit := range targetCommits {
		wg.Add(1)
		go func() {
			defer wg.Done()
			checkBranch(targetCommit, opts, progressIndicator, i)
		}()
	}
	go func() {
		wg.Wait()
		progressIndicator.Quit()
	}()
	progressIndicator.Show()
}

func checkBranch(targetCommit templates.GitLog, opts AddReviewersOptions, progressIndicator *interactive.ProgressIndicator, index int) {
	defer progressIndicator.SendErrorOnPanic()
	if opts.WhenChecksPass {
		waitForChecks(targetCommit, opts, progressIndicator, index)
	}
	markPrReady(targetCommit, progressIndicator, index)
	if opts.Reviewers != "" {
		addReviewers(targetCommit, opts.Reviewers, progressIndicator, index)
	}
	if opts.AutoMerge {
		util.ExecuteOrDie(util.ExecuteOptions{Retries: gitutil.GhRetries}, "gh", "pr", "merge", targetCommit.Branch, "--auto", "--squash", gitutil.GhRepoArgs())
		progressIndicator.SetLogLine(index, "Auto-merge enabled")
	}
}

func countdown(progressIndicator *interactive.ProgressIndicator, index int, seconds int, message string) {
	for seconds > 0 {
		unit := "seconds"
		if seconds == 1 {
			unit = "second"
		}
		progressIndicator.SetLogLine(index, fmt.Sprint("Waiting ", seconds, " ", unit, " ", message))
		util.Sleep(1 * time.Second)
		seconds--
	}
}

func waitForChecks(targetCommit templates.GitLog, opts AddReviewersOptions, progressIndicator *interactive.ProgressIndicator, index int) {
	if opts.WaitBeforePolling > 0 {
		countdown(progressIndicator, index, int(opts.WaitBeforePolling.Seconds()), "for Github to add checks to pushed changes")
	}
	for {
		summary := gitutil.GetChecksStatus(targetCommit.Branch, opts.MinChecks)
		progressIndicator.SetProgress(index, float64(summary.PercentageComplete()))
		if summary.IsFailing() {
			if !opts.Silent {
				util.ExecuteOrDie(util.ExecuteOptions{}, "say", "Checks failed")
			}
			panic(fmt.Sprint("Checks failed for ", targetCommit, ". "+
				"Total: ", summary.Total(),
				" | Passed: ", summary.Passing,
				" | Pending: ", summary.Pending,
				" | Failed: ", summary.Failing,
				"\n"))
		}
		if summary.Total() < summary.MinChecks {
			progressIndicator.SetLogLine(index, fmt.Sprint("Waiting for at least ", summary.MinChecks, " checks to be added to PR. Currently only ", summary.Total()))
		} else {
			progressIndicator.SetLogLine(index, fmt.Sprint(summary.Passing, "/", summary.Passing+summary.Pending, " checks passed"))
			if summary.IsSuccess() {
				break
			}
		}
		util.Sleep(opts.PollFrequency)
	}
}

func markPrReady(targetCommit templates.GitLog, progressIndicator *interactive.ProgressIndicator, index int) {
	progressIndicator.SetLogLine(index, "Marking PR as ready for review")
	util.ExecuteOrDie(util.ExecuteOptions{Retries: gitutil.GhRetries}, "gh", "pr", "ready", targetCommit.Branch, gitutil.GhRepoArgs())
	progressIndicator.SetLogLine(index, "PR marked as ready for review")
}

func addReviewers(targetCommit templates.GitLog, reviewers string, progressIndicator *interactive.ProgressIndicator, index int) {
	appConfig := util.GetAppConfig()
	countdown(progressIndicator, index, waitForOtherReviewersSeconds, "for any automatically assigned reviewers to be added...")
	progressIndicator.SetLogLine(index, "Checking if user has already approved latest commit")
	approvingUsers, nonApprovingUsers := getNonApprovingUsers(targetCommit, reviewers)
	if nonApprovingUsers != reviewers {
		slog.Info(fmt.Sprint("Skipping reviewers that have already approved: " + approvingUsers))
	}
	if nonApprovingUsers != "" {
		if appConfig.DemoMode {
			progressIndicator.SetLogLine(index, fmt.Sprint("Added reviewers ", nonApprovingUsers))
		} else {
			prUrl := strings.TrimSpace(
				util.ExecuteOrDie(util.ExecuteOptions{},
					"gh", "pr", "edit", targetCommit.Branch, "--add-reviewer", nonApprovingUsers, gitutil.GhRepoArgs(),
				),
			)
			progressIndicator.SetLogLine(index, fmt.Sprint("Added reviewers ", nonApprovingUsers, " to ", prUrl))
		}
	}
}

func getNonApprovingUsers(commit templates.GitLog, reviewers string) (string, string) {
	allApprovingUsers := gitutil.GetAllApprovingUsers(commit.Branch)
	approvingUsers := make([]string, 0)
	nonApprovingUsers := make([]string, 0)
	for _, reviewer := range strings.Split(reviewers, ",") {
		if slices.Contains(allApprovingUsers, reviewer) {
			approvingUsers = append(approvingUsers, reviewer)
		} else {
			nonApprovingUsers = append(nonApprovingUsers, reviewer)
		}
	}
	return strings.Join(approvingUsers, ","), strings.Join(nonApprovingUsers, ",")
}
