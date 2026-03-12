package commands

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"

	"github.com/fatih/color"

	"github.com/tinyspeck/gh-stacked-diff/v2/templates"

	"github.com/tinyspeck/gh-stacked-diff/v2/interactive"
	"github.com/tinyspeck/gh-stacked-diff/v2/util"
)

func createNewCommand() Command {
	flagSet := flag.NewFlagSet("new", flag.ContinueOnError)

	draft := flagSet.Bool("draft", true, "Whether to create the PR as draft")
	featureFlag := flagSet.String("feature-flag", "", "Value for FEATURE_FLAG in PR description")
	baseBranch := flagSet.String("base", "", "Base branch for Pull Request. Default is "+util.GetMainBranchForHelp())

	reviewers, silent, minChecks, merge := addReviewersFlags(flagSet)

	indicatorTypeString := addIndicatorFlag(flagSet)

	return Command{
		FlagSet: flagSet,
		Summary: "Create a new pull request from a commit on main",
		Description: "Create a new PR with a cherry-pick of the given commit indicator.\n" +
			"\n" +
			"This command first creates an associated branch, (with a name based\n" +
			"on the commit summary), and then uses Github CLI to create a PR.\n" +
			"\n" +
			"Can also add reviewers once PR checks have passed, see \"--reviewers\" flag.",
		Usage: "sd new [flags] [commitIndicator]\n" +
			"\n" +
			"If commitIndicator is missing then you will be prompted to select commit:\n" +
			"\n" +
			"   [enter]    confirms selection\n" +
			"   [up,k]     moves cursor up\n" +
			"   [down,j]   moves cursor down\n" +
			"   [q,esc]    cancels\n" +
			"\n" +
			color.HiWhiteString("Ticket Number:") + "\n" +
			"\n" +
			"If you prefix a (Jira-like formatted) ticket number to the git commit\n" +
			"summary then the \"Ticket\" section of the PR description will be \n" +
			"populated with it.\n" +
			"\n" +
			"For example:\n" +
			"\n" +
			"\"CONV-9999 Add new feature\"\n" +
			"\n" +
			color.HiWhiteString("Templates:") + "\n" +
			"\n" +
			"The Pull Request Title, Body (aka Description), and Branch Name are\n" +
			"created from golang templates.\n" +
			"\n" +
			"The default templates are:\n" +
			"\n" +
			"   branch-name.template:      templates/config/branch-name.template\n" +
			"   pr-description.template:   templates/config/pr-description.template\n" +
			"   pr-title.template:         templates/config/pr-title.template\n" +
			"\n" +
			"To change a template, copy the default from templates/config/ into\n" +
			"~/.gh-stacked-diff/ and modify contents.\n" +
			"\n" +
			"The possible values for the templates are:\n" +
			"\n" +
			"   CommitBody                   Body of the commit message\n" +
			"   CommitSummary                Summary line of the commit message\n" +
			"   CommitSummaryCleaned         Summary line of the commit message without\n" +
			"                                spaces or special characters\n" +
			"   CommitSummaryWithoutTicket   Summary line of the commit message without\n" +
			"                                the prefix of the ticket number\n" +
			"   FeatureFlag                  Value passed to feature-flag flag\n" +
			"   TicketNumber                 Jira ticket as parsed from the commit summary\n" +
			"   Username                     Name as parsed from git config email.\n" +
			"   UsernameCleaned              Username with dots (.) converted to dashes (-).\n",
		OnSelected: func(appConfig util.AppConfig, command Command) {
			if flagSet.NArg() > 1 {
				commandError(appConfig, flagSet, "too many arguments", command.Usage)
			}
			util.RequireMainBranch()
			selectCommitOptions := interactive.CommitSelectionOptions{
				Prompt:      "What commit do you want to create a PR from?",
				CommitType:  interactive.CommitTypeNoPr,
				MultiSelect: false,
			}
			targetCommits := getTargetCommits(appConfig, command, flagSet.Args(), indicatorTypeString, selectCommitOptions)
			// Note: set the default here rather than via flags to avoid GetMainBranchOrDie being called before OnSelected.
			if *baseBranch == "" {
				*baseBranch = util.GetMainBranchOrDie()
			}
			markReady := promptForReviewers(appConfig, reviewers, flagSet.NArg() == 0 && *draft)
			createNewPr(*draft, *featureFlag, *baseBranch, targetCommits[0])
			if *reviewers != "" || markReady {
				addReviewersToPr(appConfig, targetCommits, AddReviewersOptions{
					WhenChecksPass: true,
					Silent:         *silent,
					MinChecks:      *minChecks,
					Reviewers:      *reviewers,
					PollFrequency:  DefaultPollFrequency,
					AutoMerge:      *merge,
				})
			}
		}}
}

// Creates a new pull request via Github CLI.
func createNewPr(draft bool, featureFlag string, baseBranch string, gitLog templates.GitLog) {
	templates.RequireCommitOnMain(gitLog.Commit)
	util.WithStashAndRollback("sd new "+gitLog.Commit+" "+gitLog.Subject, func(rollbackManager *util.GitRollbackManager) {
		createBranchAndCherryPick(rollbackManager, baseBranch, gitLog)
		pushAndCreateGhPr(draft, featureFlag, baseBranch, gitLog)
		rollbackManager.Clear()
		openPrAndSwitchBack(gitLog)
	})
}

func createBranchAndCherryPick(rollbackManager *util.GitRollbackManager, baseBranch string, gitLog templates.GitLog) {
	var commitToBranchFrom string
	if baseBranch == util.GetMainBranchOrDie() {
		commitToBranchFrom = util.FirstOriginMainCommit(util.GetMainBranchOrDie())
		slog.Info(fmt.Sprint("Switching to branch ", gitLog.Branch, " based off commit ", commitToBranchFrom))
	} else {
		commitToBranchFrom = baseBranch
		slog.Info(fmt.Sprint("Switching to branch ", gitLog.Branch, " based off branch ", baseBranch))
	}
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "--no-track", gitLog.Branch, commitToBranchFrom)
	rollbackManager.CreatedBranch(gitLog.Branch)
	util.GitSwitch(gitLog.Branch)
	slog.Info(fmt.Sprint("Cherry picking ", gitLog.Commit))
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "cherry-pick", gitLog.Commit)
}

func pushAndCreateGhPr(draft bool, featureFlag string, baseBranch string, gitLog templates.GitLog) {
	slog.Info("Pushing to remote")
	// -u is required because in newer versions of Github CLI the upstream must be set.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-c", "push.default=current", "push", "--force-with-lease", "-u")
	prText := templates.GetPullRequestText(gitLog.Commit, featureFlag)
	slog.Info("Creating PR via gh")
	createPrOutput := createPr(prText, baseBranch, draft)
	slog.Info(fmt.Sprint("Created PR ", createPrOutput))
}

func openPrAndSwitchBack(gitLog templates.GitLog) {
	util.ExecuteOrDie(util.ExecuteOptions{Retries: util.GhRetries}, "gh", "pr", "view", "--web")
	slog.Info(fmt.Sprint("Switching back to " + util.GetMainBranchOrDie()))
	util.GitSwitch(util.GetMainBranchOrDie())
	// Suppress the "use --reapply-cherry-picks" hint which is not appropriate for stacked diff workflow.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "config", "advice.skippedCherryPicks", "false")
}

func createPr(prText templates.PullRequestText, baseBranch string, draft bool) string {
	createPrArgsNoDraft := []string{"pr", "create", "--title", prText.Title, "--body", prText.Description, "--fill", "--base", baseBranch}
	createPrArgs := createPrArgsNoDraft
	if draft {
		createPrArgs = append(createPrArgs, "--draft")
	}
	createPrOutput, createPrErr := util.Execute(util.ExecuteOptions{}, "gh", createPrArgs...)
	if createPrErr != nil {
		if draft && strings.Contains(createPrOutput, "Draft pull requests are not supported") {
			slog.Warn("Draft PRs not supported, trying again without draft.\nUse \"--draft=false\" to avoid this warning.")
			return util.ExecuteOrDie(util.ExecuteOptions{Retries: util.GhRetries}, "gh", createPrArgsNoDraft...)
		} else {
			slog.Warn("Retrying: " + "\"gh " + strings.Join(createPrArgs, " ") + "\": " + createPrErr.Error())
			util.Sleep(util.RetryDelay)
			return util.ExecuteOrDie(util.ExecuteOptions{Retries: util.GhRetries - 1}, "gh", createPrArgs...)
		}
	} else {
		return createPrOutput
	}
}
