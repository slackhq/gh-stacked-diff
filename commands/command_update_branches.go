package commands

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createUpdateBranchesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update-branches",
		Short: "Update PR branches with contents of the local " + gitutil.GetMainBranchForHelp() + " commit",
		Long: "Update PR branches so they match the contents of local " + gitutil.GetMainBranchForHelp() + ".\n" +
			"\n" +
			"Shows a selection dialog with PR branches whose diffs differ from the\n" +
			"commit on local " + gitutil.GetMainBranchForHelp() + ". Branches already in sync are not selectable.\n" +
			"\n" +
			"Draft PRs: Branch is recreated from origin/" + gitutil.GetMainBranchForHelp() + " with the commit\n" +
			"cherry-picked on top, then force-pushed.\n" +
			"\n" +
			"Non-draft PRs: origin/" + gitutil.GetMainBranchForHelp() + " is merged into the branch, creating\n" +
			"a merge commit, then pushed.",
		Args: cobra.NoArgs,
		Annotations: map[string]string{
			checkRepoAnnotation: "true",
		},
		Run: func(cmd *cobra.Command, args []string) {
			updateBranches()
		},
	}
}

func updateBranches() {
	gitutil.RequireMainBranch()
	newCommits := templates.GetNewCommits("HEAD", "")
	if len(newCommits) == 0 {
		slog.Info("No commits ahead of origin/" + gitutil.GetRemoteMainBranchOrDie())
		return
	}
	branchNames := make([]string, len(newCommits))
	for i, log := range newCommits {
		branchNames[i] = log.Branch
	}
	prBranches := gitutil.CheckLocalBranches("", branchNames)

	disabledBranches := make(map[string]bool)
	hasEnabledBranch := false
	for _, commit := range newCommits {
		if !slices.Contains(prBranches, commit.Branch) {
			continue
		}
		if !branchNeedsDiffUpdate(commit) {
			disabledBranches[commit.Branch] = true
		} else {
			hasEnabledBranch = true
		}
	}
	if !hasEnabledBranch {
		slog.Info("All PR branches are already in sync")
		return
	}

	selectedCommits, err := interactive.GetCommitSelection(interactive.CommitSelectionOptions{
		CommitType:       interactive.CommitTypePr,
		MultiSelect:      true,
		Prompt:           "Select PR branches to update with " + gitutil.GetLocalMainBranchOrDie() + ":",
		DisabledBranches: disabledBranches,
	})
	if err != nil || len(selectedCommits) == 0 {
		return
	}

	mainBranch := gitutil.GetLocalMainBranchOrDie()
	for _, commit := range selectedCommits {
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Warn(fmt.Sprint("Failed to update branch ", commit.Branch, ": ", r))
					gitutil.GitSwitch(mainBranch)
				}
			}()
			updatePrBranch(commit, mainBranch)
		}()
	}
}

func branchNeedsDiffUpdate(commit templates.GitLog) bool {
	commitDiff := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "diff", commit.Commit+"~1", commit.Commit)
	mergeBase := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "merge-base", "origin/"+gitutil.GetRemoteMainBranchOrDie(), commit.Branch)
	branchDiff := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "diff", mergeBase, commit.Branch)
	return commitDiff != branchDiff
}

func updatePrBranch(commit templates.GitLog, mainBranch string) {
	appConfig := util.GetAppConfig()
	prStatus := gitutil.GetPullRequestStatus(commit.Branch, 0)
	if prStatus.IsDraft {
		updateDraftBranch(commit, mainBranch, appConfig)
	} else {
		updateNonDraftBranch(commit, mainBranch, appConfig)
	}
}

func updateDraftBranch(commit templates.GitLog, mainBranch string, appConfig util.AppConfig) {
	branch := commit.Branch
	slog.Info(fmt.Sprint("Updating draft PR branch: ", branch))
	remoteBranch := "origin/" + gitutil.GetRemoteMainBranchOrDie()
	util.ExecuteOrDie(util.ExecuteOptions{Io: appConfig.Io}, "git", "branch", "-f", branch, remoteBranch)
	gitutil.GitSwitch(branch)
	if _, err := gitutil.CherryPick(util.ExecuteOptions{Io: appConfig.Io}, commit.Commit); err != nil {
		slog.Warn(fmt.Sprint("Cherry-pick failed for ", branch, ". Skipping update for this branch."))
		// nolint:errcheck
		util.Execute(util.ExecuteOptions{}, "git", "cherry-pick", "--abort")
		gitutil.GitSwitch(mainBranch)
		return
	}
	gitutil.GitPushOrDie(util.ExecuteOptions{}, "push", "--force-with-lease", "origin", branch+":"+branch)
	gitutil.GitSwitch(mainBranch)
	slog.Info(fmt.Sprint("Updated draft PR branch: ", branch))
}

func updateNonDraftBranch(commit templates.GitLog, mainBranch string, appConfig util.AppConfig) {
	branch := commit.Branch
	slog.Info(fmt.Sprint("Updating PR branch: ", branch))
	remoteBranch := "origin/" + gitutil.GetRemoteMainBranchOrDie()
	gitutil.GitSwitch(branch)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "fetch", "origin", branch)
	if _, err := util.Execute(util.ExecuteOptions{Io: appConfig.Io}, "git", "merge", "--ff-only", "origin/"+branch); err != nil {
		slog.Warn(fmt.Sprint("Could not fast-forward ", branch, " to remote. Continuing with merge."))
	}
	if _, err := util.Execute(util.ExecuteOptions{Io: appConfig.Io}, "git", "merge", remoteBranch); err != nil {
		slog.Warn(fmt.Sprint("Merge conflict updating ", branch, ". Skipping update for this branch."))
		// nolint:errcheck
		util.Execute(util.ExecuteOptions{}, "git", "merge", "--abort")
		gitutil.GitSwitch(mainBranch)
		return
	}
	gitutil.GitPushOrDie(util.ExecuteOptions{}, "push", "origin", branch+":"+branch)
	gitutil.GitSwitch(mainBranch)
	slog.Info(fmt.Sprint("Updated PR branch: ", branch))
}
