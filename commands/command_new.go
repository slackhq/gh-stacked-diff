package commands

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/fatih/color"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/templates"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createNewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new [commitIndicator]",
		Short: "Create a new pull request from a commit on main",
		Long: "Create a new PR with a cherry-pick of the given commit indicator.\n" +
			"\n" +
			"This command first creates an associated branch, (with a name based\n" +
			"on the commit summary), and then uses Github CLI to create a PR.\n" +
			"\n" +
			"Can also add reviewers once PR checks have passed, see \"--reviewers\" flag.\n" +
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
			"   TicketUrlPattern             URL for the ticket, with {TicketNumber}\n" +
			"                                replaced by the actual ticket number.\n" +
			"                                Configured via config.yaml or --config.\n" +
			"   Username                     Name as parsed from git config email.\n" +
			"   UsernameCleaned              Username with dots (.) converted to dashes (-).\n",
		Args: cobra.MaximumNArgs(1),
		Annotations: map[string]string{
			checkRepoAnnotation: "true",
		},
	}

	draft := cmd.Flags().BoolP("draft", "d", true, "Whether to create the PR as draft")
	noTemplate := cmd.Flags().BoolP("no-template", "T", false, "Use the commit body as the PR description without applying the PR description template")
	featureFlag := cmd.Flags().StringP("feature-flag", "f", "", "Value for FEATURE_FLAG in PR description")
	baseBranch := cmd.Flags().StringP("base", "b", "", "Base branch for Pull Request. Default is "+gitutil.GetMainBranchForHelp())
	_ = cmd.RegisterFlagCompletionFunc("base", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		branches := strings.Fields(util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--format=%(refname:short)"))
		return branches, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.MarkFlagsMutuallyExclusive("no-template", "feature-flag")

	reviewers, silent, minChecks, merge := addReviewersFlags(cmd)
	indicatorTypeString := addIndicatorFlag(cmd)

	cmd.Run = func(cmd *cobra.Command, args []string) {
		gitutil.RequireMainBranch()
		userConfig := util.GetUserConfig()
		if !cmd.Flags().Changed("no-template") {
			*noTemplate = userConfig.NoTemplate
		}
		selectCommitOptions := interactive.CommitSelectionOptions{
			Prompt:      "What commit do you want to create a PR from?",
			CommitType:  interactive.CommitTypeNoPr,
			MultiSelect: false,
		}
		targetCommits := getTargetCommits(args, indicatorTypeString, selectCommitOptions)
		// Note: set the default here rather than via flags to avoid GetLocalMainBranchOrDie being called before Run.
		var remoteBaseBranch string
		if *baseBranch == "" {
			*baseBranch = gitutil.GetLocalMainBranchOrDie()
			remoteBaseBranch = gitutil.GetRemoteMainBranchOrDie()
		} else {
			util.RequireGitRef(*baseBranch)
			remoteBaseBranch = *baseBranch
		}
		ticketUrlPattern := userConfig.TicketUrlPattern
		if !*noTemplate && ticketUrlPattern == "" && templates.HasTicketNumber(targetCommits[0].Subject) && templates.TemplateUsesTicketUrlPattern() {
			ticketUrlPattern = interactive.PromptForStringOrDie(
				"Ticket URL pattern (use {TicketNumber} as placeholder):",
				util.ExampleTicketUrlPattern,
				[]string{util.ExampleTicketUrlPattern},
			)
			util.SaveTicketUrlPattern(ticketUrlPattern)
		}
		selectedReviewers, markReady := promptForReviewers(len(args) == 0 && *draft && *reviewers == "", userConfig, *merge)
		createNewPr(*draft, *noTemplate, *featureFlag, ticketUrlPattern, *baseBranch, remoteBaseBranch, targetCommits[0])
		maybeAddReviewers(*reviewers, selectedReviewers, markReady, targetCommits, AddReviewersOptions{
			WhenChecksPass: true,
			Silent:         *silent,
			MinChecks:      *minChecks,
			PollFrequency:  userConfig.PollInterval,
			AutoMerge:      *merge,
		})
	}

	return cmd
}

// Creates a new pull request via Github CLI.
func createNewPr(draft bool, noTemplate bool, featureFlag string, ticketUrlPattern string, baseBranch string, remoteBaseBranch string, gitLog templates.GitLog) {
	templates.RequireCommitOnMain(gitLog.Commit)
	gitutil.WithStashAndRollback("sd new "+gitLog.Commit+" "+gitLog.Subject, func(rollbackManager *gitutil.GitRollbackManager) {
		createBranchAndCherryPick(rollbackManager, baseBranch, gitLog)
		pushAndCreateGhPr(draft, noTemplate, featureFlag, ticketUrlPattern, remoteBaseBranch, gitLog)
		rollbackManager.Clear()
		openPrAndSwitchBack(gitLog)
	})
}

func createBranchAndCherryPick(rollbackManager *gitutil.GitRollbackManager, baseBranch string, gitLog templates.GitLog) {
	var commitToBranchFrom string
	if baseBranch == gitutil.GetLocalMainBranchOrDie() {
		commitToBranchFrom = gitutil.GetMergeBaseWithOriginMain(gitutil.GetLocalMainBranchOrDie())
		slog.Info(fmt.Sprint("Switching to branch ", gitLog.Branch, " based off commit ", commitToBranchFrom))
	} else {
		commitToBranchFrom = baseBranch
		slog.Info(fmt.Sprint("Switching to branch ", gitLog.Branch, " based off branch ", baseBranch))
	}
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "--no-track", gitLog.Branch, commitToBranchFrom)
	rollbackManager.CreatedBranch(gitLog.Branch)
	gitutil.GitSwitch(gitLog.Branch)
	slog.Info(fmt.Sprint("Cherry picking ", gitLog.Commit))
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "cherry-pick", gitLog.Commit)
}

func pushAndCreateGhPr(draft bool, noTemplate bool, featureFlag string, ticketUrlPattern string, remoteBaseBranch string, gitLog templates.GitLog) {
	slog.Info("Pushing to remote")
	// -u is required because in newer versions of Github CLI the upstream must be set.
	gitutil.GitPushOrDie(util.ExecuteOptions{}, "-c", "push.default=current", "push", "--force-with-lease", "-u")
	var prText templates.PullRequestText
	if noTemplate {
		prText = templates.GetPullRequestTextRaw(gitLog.Commit)
	} else {
		prText = templates.GetPullRequestText(gitLog.Commit, featureFlag, ticketUrlPattern)
	}
	slog.Info("Creating PR via gh")
	createPrOutput := createPr(prText, remoteBaseBranch, draft)
	slog.Info(fmt.Sprint("Created PR ", createPrOutput))
}

func openPrAndSwitchBack(gitLog templates.GitLog) {
	if _, err := util.Execute(util.ExecuteOptions{Retries: gitutil.GhRetries}, "gh", "pr", "view", "--web", gitutil.GhRepoArgs(), gitLog.Branch); err != nil {
		slog.Warn("Could not view PR: " + err.Error())
	}
	slog.Info(fmt.Sprint("Switching back to " + gitutil.GetLocalMainBranchOrDie()))
	gitutil.GitSwitch(gitutil.GetLocalMainBranchOrDie())
	// Suppress the "use --reapply-cherry-picks" hint which is not appropriate for stacked diff workflow.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "config", "advice.skippedCherryPicks", "false")
}

func createPr(prText templates.PullRequestText, remoteBaseBranch string, draft bool) string {
	baseArgs := []string{"pr", "create", "--title", prText.Title, "--body", prText.Description,
		"--fill", "--base", remoteBaseBranch}
	var draftArgs []string
	if draft {
		draftArgs = append(baseArgs, "--draft")
	} else {
		draftArgs = baseArgs
	}
	createPrOutput, createPrErr := util.Execute(util.ExecuteOptions{}, "gh", draftArgs, gitutil.GhRepoArgs())
	if createPrErr != nil {
		if draft && strings.Contains(createPrOutput, "Draft pull requests are not supported") {
			slog.Warn("Draft PRs not supported, trying again without draft.\nUse \"--draft=false\" to avoid this warning.")
			return util.ExecuteOrDie(util.ExecuteOptions{Retries: gitutil.GhRetries}, "gh", baseArgs, gitutil.GhRepoArgs())
		} else {
			firstLine, _, _ := strings.Cut(strings.Join(draftArgs, " "), "\n")
			slog.Warn("Retrying: " + "\"gh " + firstLine + "\": " + createPrErr.Error())
			util.Sleep(util.RetryDelay)
			return util.ExecuteOrDie(util.ExecuteOptions{Retries: gitutil.GhRetries - 1}, "gh", draftArgs, gitutil.GhRepoArgs())
		}
	} else {
		return createPrOutput
	}
}
