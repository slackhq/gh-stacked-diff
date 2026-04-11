package commands

import (
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

const waitForAddedChecksSeconds = 10

func createUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [PR commitIndicator [fixup commitIndicator...]]",
		Short: "Add commits from " + gitutil.GetMainBranchForHelp() + " to an existing PR",
		Long: "Add commits from local " + gitutil.GetMainBranchForHelp() + " branch to an existing PR.\n" +
			"\n" +
			"Can also add reviewers once PR checks have passed, see \"--reviewers\" flag.\n" +
			"\n" +
			"Note: git hooks (pre-commit, etc.) are bypassed during the local rebase. This\n" +
			"is safe because the resulting commits are cherry-picked onto the PR branch and\n" +
			"pushed, which runs hooks normally.",
		Annotations: map[string]string{
			checkRepoAnnotation: "true",
		},
	}
	indicatorTypeString := addIndicatorFlag(cmd)
	reviewers, silent, minChecks, merge := addReviewersFlags(cmd)
	cmd.Run = func(cmd *cobra.Command, args []string) {
		gitutil.RequireMainBranch()
		userConfig := util.GetUserConfig()
		destCommit := getDestCommit(args, indicatorTypeString)
		commitsToCherryPick := getCommitsToCherryPick(args, indicatorTypeString)
		selectedReviewers, markReady := promptForReviewers(len(args) < 2 && *reviewers == "", userConfig, *merge)
		updatePr(destCommit, commitsToCherryPick)
		maybeAddReviewers(*reviewers, selectedReviewers, markReady, []templates.GitLog{destCommit}, AddReviewersOptions{
			WhenChecksPass:    true,
			Silent:            *silent,
			MinChecks:         *minChecks,
			PollFrequency:     userConfig.PollInterval,
			AutoMerge:         *merge,
			WaitBeforePolling: waitForAddedChecksSeconds * time.Second,
		})
	}
	return cmd
}

func getDestCommit(args []string, indicatorTypeString *string) templates.GitLog {
	selectPrOptions := interactive.CommitSelectionOptions{
		Prompt:      "What PR do you want to update?",
		CommitType:  interactive.CommitTypePr,
		MultiSelect: false,
	}
	targetCommits := getTargetCommits(args[:min(len(args), 1)], indicatorTypeString, selectPrOptions)
	return targetCommits[0]
}

func getCommitsToCherryPick(args []string, indicatorTypeString *string) []templates.GitLog {
	selectCommitsOptions := interactive.CommitSelectionOptions{
		Prompt:      "What commits do you want to add?",
		CommitType:  interactive.CommitTypeNoPr,
		MultiSelect: true,
	}
	var commitsFromCommandLine []string
	if len(args) > 1 {
		commitsFromCommandLine = args[1:]
	}
	return getTargetCommits(commitsFromCommandLine, indicatorTypeString, selectCommitsOptions)
}

// Add commits from main to an existing PR.
func updatePr(destCommit templates.GitLog, commitsToCherryPick []templates.GitLog) {
	appConfig := util.GetAppConfig()
	templates.RequireCommitOnMain(destCommit.Commit)
	checkNotMerged(destCommit.Branch)
	gitutil.WithStashAndRollback("before update-pr "+destCommit.Commit+" "+destCommit.Subject, func(rollbackManager *gitutil.GitRollbackManager) {
		gitutil.GitSwitch(destCommit.Branch)
		rollbackManager.SaveState() // Save state again for associated branch.
		slog.Info("Fast forwarding in case there were any commits made via github web interface")
		util.ExecuteOrDie(util.ExecuteOptions{}, "git", "fetch", "origin", destCommit.Branch)
		forcePush := false
		if _, err := util.Execute(util.ExecuteOptions{Io: appConfig.Io}, "git", "merge", "--ff-only", "origin/"+destCommit.Branch); err != nil {
			slog.Info(fmt.Sprint("Could not fast forward to match origin. Rebasing instead. ", err))
			gitutil.RebaseAndSkipAllEmptyOrDie(util.ExecuteOptions{Io: appConfig.Io}, "origin", destCommit.Branch)
			// As we rebased, a force push may be required.
			forcePush = true
		}

		slog.Info(fmt.Sprint("Cherry picking ", commitsToCherryPick))
		cherryPickArgs := make([]string, 1+len(commitsToCherryPick))
		cherryPickArgs[0] = "cherry-pick"
		for i, commit := range commitsToCherryPick {
			cherryPickArgs[i+1] = commit.Commit
		}
		_, cherryPickError := util.Execute(util.ExecuteOptions{}, "git", cherryPickArgs...)
		if cherryPickError != nil {
			slog.Info("First attempt at cherry-pick failed")
			util.ExecuteOrDie(util.ExecuteOptions{}, "git", "cherry-pick", "--abort")
			rebaseCommit := gitutil.GetMergeBaseWithOriginMain(gitutil.GetLocalMainBranchOrDie())
			slog.Info(fmt.Sprint("Rebasing with the base commit on "+gitutil.GetLocalMainBranchOrDie()+" branch, ", rebaseCommit,
				", in case the local "+gitutil.GetLocalMainBranchOrDie()+" was rebased with origin/"+gitutil.GetRemoteMainBranchOrDie()))
			gitutil.RebaseAndSkipAllEmptyOrDie(util.ExecuteOptions{Io: appConfig.Io}, rebaseCommit)
			slog.Info(fmt.Sprint("Cherry picking again ", commitsToCherryPick))
			util.ExecuteOrDie(util.ExecuteOptions{Io: appConfig.Io}, "git", cherryPickArgs...)
			forcePush = true
		}
		slog.Info("Switching back to " + gitutil.GetLocalMainBranchOrDie())
		gitutil.GitSwitch(gitutil.GetLocalMainBranchOrDie())
		slog.Info(fmt.Sprint("Rebasing, marking as fixup ", commitsToCherryPick, " for target ", destCommit.Commit))
		commitHashes := util.MapSlice(commitsToCherryPick, func(commit templates.GitLog) string {
			return commit.Commit
		})
		environmentVariables := []string{
			sequenceEditorEnvVar("sequence-editor-mark-as-fixup", append([]string{destCommit.Commit}, commitHashes...)...),
		}
		slog.Debug(fmt.Sprint("Using sequence editor ", environmentVariables))
		options := util.ExecuteOptions{EnvironmentVariables: environmentVariables, Io: appConfig.Io}
		rebaseBase := earliestCommit(destCommit, commitsToCherryPick)
		gitutil.RebaseAndSkipAllEmptyOrDie(options, "-i", rebaseBase+"^")
		// Do the push last so that if there is a rollback origin was not updated.
		slog.Info("Pushing to remote")
		if forcePush {
			if _, err := util.Execute(util.ExecuteOptions{}, "git", "push", "origin", destCommit.Branch+":"+destCommit.Branch); err != nil {
				slog.Info("Regular push failed, force pushing instead.")
				util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "--force-with-lease", "origin", destCommit.Branch+":"+destCommit.Branch)
			}
		} else {
			util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", destCommit.Branch+":"+destCommit.Branch)
		}
		rollbackManager.Clear()
	})
}

// earliestCommit returns the commit hash of the oldest commit among destCommit
// and commitsToCherryPick. GetNewCommits returns newest-first, so the highest
// index is the oldest commit.
func earliestCommit(destCommit templates.GitLog, commitsToCherryPick []templates.GitLog) string {
	newCommits := templates.GetNewCommits("HEAD", "")
	earliest := destCommit.Commit
	earliestIdx := slices.IndexFunc(newCommits, func(gl templates.GitLog) bool {
		return gl.Commit == destCommit.Commit
	})
	for _, cp := range commitsToCherryPick {
		idx := slices.IndexFunc(newCommits, func(gl templates.GitLog) bool {
			return gl.Commit == cp.Commit
		})
		if idx > earliestIdx {
			earliestIdx = idx
			earliest = cp.Commit
		}
	}
	return earliest
}

func checkNotMerged(branchName string) {
	slog.Info("Checking if PR was already merged...")
	mergedBranches := getMergedBranches()
	if slices.Contains(mergedBranches, branchName) {
		interactive.ConfirmOrDie("It looks like this PR was already merged. Try running \"sd rebase-main\". Are you sure you want to update this PR?", false)
	}
}
