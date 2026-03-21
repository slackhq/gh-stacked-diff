package commands

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

const DefaultPollFrequency = 30 * time.Second

func createAddReviewersCommand(appConfig util.AppConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-reviewers [commitIndicator...]",
		Short: "Add reviewers to Pull Request on Github once its checks have passed",
		Long: "Add reviewers to Pull Request on Github once its checks have passed.\n" +
			"\n" +
			"If PR is marked as a Draft, it is first marked as \"Ready for Review\".",
	}
	indicatorTypeString := addIndicatorFlag(cmd)

	whenChecksPass := cmd.Flags().BoolP("when-checks-pass", "w", true, "Poll until all checks pass before adding reviewers")
	pollFrequency := cmd.Flags().DurationP("poll-frequency", "p", DefaultPollFrequency,
		"Frequency which to poll checks. For valid formats see https://pkg.go.dev/time#ParseDuration")
	reviewers, silent, minChecks, merge := addReviewersFlags(cmd, appConfig)

	cmd.Run = func(cmd *cobra.Command, args []string) {
		selectPrsOptions := interactive.CommitSelectionOptions{
			Prompt:      "What PR do you want to add reviewers too?",
			CommitType:  interactive.CommitTypePr,
			MultiSelect: true,
		}
		targetCommits := getTargetCommits(appConfig, args, indicatorTypeString, selectPrsOptions)
		// Reverse the order as getTargetCommits returns cherry-pick order and we want to display in log order.
		slices.Reverse(targetCommits)
		if *reviewers == "" {
			*reviewers = interactive.UserSelection(appConfig)
		}
		if *reviewers != "" {
			slog.Info("Using reviewers " + *reviewers)
			interactive.ReviewersHistory.AddToHistory(appConfig, *reviewers)
		}
		addReviewersToPr(appConfig, targetCommits, AddReviewersOptions{
			WhenChecksPass: *whenChecksPass,
			Silent:         *silent,
			MinChecks:      *minChecks,
			Reviewers:      *reviewers,
			PollFrequency:  *pollFrequency,
			AutoMerge:      *merge,
		})
	}
	return cmd
}

type AddReviewersOptions struct {
	WhenChecksPass bool
	Silent         bool
	MinChecks      int
	Reviewers      string
	PollFrequency  time.Duration
	AutoMerge      bool
}

// Adds reviewers to a PR once checks have passed via Github CLI.
func addReviewersToPr(appConfig util.AppConfig, targetCommits []templates.GitLog, opts AddReviewersOptions) {
	progressIndicatorMessages := util.MapSlice(targetCommits, func(next templates.GitLog) string {
		return next.String()
	})
	progressIndicator := interactive.NewProgressIndicator(appConfig.Io, progressIndicatorMessages)
	var wg sync.WaitGroup
	for i, targetCommit := range targetCommits {
		wg.Add(1)
		go func() {
			defer wg.Done()
			checkBranch(appConfig, targetCommit, opts, progressIndicator, i)
		}()
	}
	go func() {
		wg.Wait()
		progressIndicator.Quit()
	}()
	progressIndicator.Show(appConfig)
}

func checkBranch(appConfig util.AppConfig, targetCommit templates.GitLog, opts AddReviewersOptions, progressIndicator *interactive.ProgressIndicator, index int) {
	defer progressIndicator.SendErrorOnPanic()
	if opts.WhenChecksPass {
		for {
			summary := util.GetChecksStatus(appConfig, targetCommit.Branch, opts.MinChecks)
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
	progressIndicator.SetLogLine(index, "Marking PR as ready for review")
	util.ExecuteOrDie(util.ExecuteOptions{Retries: util.GhRetries}, "gh", "pr", "ready", targetCommit.Branch)
	progressIndicator.SetLogLine(index, "PR marked as ready for review")
	if opts.Reviewers != "" {
		progressIndicator.SetLogLine(index, "Waiting 10 seconds for any automatically assigned reviewers to be added...")
		util.Sleep(10 * time.Second)
		progressIndicator.SetLogLine(index, "Checking if user has already approved latest commit")
		approvingUsers, nonApprovingUsers := getNonApprovingUsers(targetCommit, opts.Reviewers)
		if nonApprovingUsers != opts.Reviewers {
			slog.Info(fmt.Sprint("Skipping reviewers that have already approved: " + approvingUsers))
		}
		if len(nonApprovingUsers) > 0 {
			if appConfig.DemoMode {
				progressIndicator.SetLogLine(index, fmt.Sprint("Added reviewers ", nonApprovingUsers))
			} else {
				prUrl := strings.TrimSpace(
					util.ExecuteOrDie(util.ExecuteOptions{},
						"gh", "pr", "edit", targetCommit.Branch, "--add-reviewer", nonApprovingUsers,
					),
				)
				progressIndicator.SetLogLine(index, fmt.Sprint("Added reviewers ", nonApprovingUsers, " to ", prUrl))
			}
		}
	}
	if opts.AutoMerge {
		util.ExecuteOrDie(util.ExecuteOptions{Retries: util.GhRetries}, "gh", "pr", "merge", targetCommit.Branch, "--auto", "--squash")
		progressIndicator.SetLogLine(index, "Auto-merge enabled")
	}
}

func getNonApprovingUsers(commit templates.GitLog, reviewers string) (string, string) {
	allApprovingUsers := util.GetAllApprovingUsers(commit.Branch)
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
