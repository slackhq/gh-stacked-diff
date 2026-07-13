package commands

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

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
			"This is the opposite of `replace-commit`. It updates PR branches to the contents of the corresponding \n" +
			"commits on local main and pushes to remote. Use this after `sd rebase-main` to update your PR \n" +
			"branches. Uses `git rebase` if the PR is in draft status, or `git merge` if the PR is open.\n" +
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

	mainBranch := gitutil.GetLocalMainBranchOrDie()
	disabledBranches := make(map[string]bool)
	hasEnabledBranch := false
	for _, commit := range newCommits {
		if !slices.Contains(prBranches, commit.Branch) {
			continue
		}
		if !branchNeedsUpdate(commit, mainBranch) {
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
		Prompt:           "Select PR branches to update with " + mainBranch + ":",
		DisabledBranches: disabledBranches,
	})
	if err != nil || len(selectedCommits) == 0 {
		return
	}
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

func branchNeedsUpdate(commit templates.GitLog, mainBranch string) bool {
	commitDiff := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "diff", "--binary", commit.Commit+"~1", commit.Commit)
	mergeBase := gitutil.GetMergeBaseWithOriginMain(commit.Branch)
	branchDiff := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "diff", "--binary", mergeBase, commit.Branch)
	if commitDiff != branchDiff {
		return true
	}
	mainMergeBase := gitutil.GetMergeBaseWithOriginMain(mainBranch)
	return !gitutil.IsAncestor(mainMergeBase, commit.Branch)
}

func updatePrBranch(commit templates.GitLog, mainBranch string) {
	appConfig := util.GetAppConfig()
	prStatus := gitutil.GetPullRequestStatus(commit.Branch, 0)
	if prStatus.IsDraft {
		updateWithRebase(commit, mainBranch, appConfig)
	} else {
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Warn(fmt.Sprint("Merge failed for ", commit.Branch, ": ", r, ". Falling back to rebase."))
					// nolint:errcheck
					util.Execute(util.ExecuteOptions{}, "git", "merge", "--abort")
					// nolint:errcheck
					util.Execute(util.ExecuteOptions{}, "git", "reset", "--hard")
					gitutil.GitSwitch(mainBranch)
					updateWithRebase(commit, mainBranch, appConfig)
				}
			}()
			updateWithMerge(commit, mainBranch, appConfig)
		}()
	}
}

func updateWithRebase(commit templates.GitLog, mainBranch string, appConfig util.AppConfig) {
	branch := commit.Branch
	slog.Info(fmt.Sprint("Updating draft PR branch: ", branch))
	mergeBase := gitutil.GetMergeBaseWithOriginMain(mainBranch)
	util.ExecuteOrDie(util.ExecuteOptions{Io: appConfig.Io}, "git", "branch", "-f", branch, mergeBase)
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

func updateWithMerge(commit templates.GitLog, mainBranch string, appConfig util.AppConfig) {
	branch := commit.Branch
	slog.Info(fmt.Sprint("Updating PR branch: ", branch))
	mergeBase := gitutil.GetMergeBaseWithOriginMain(mainBranch)
	gitutil.GitSwitch(branch)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "fetch", "origin", branch)
	util.ExecuteOrDie(util.ExecuteOptions{Io: appConfig.Io}, "git", "merge", "origin/"+branch)
	if _, err := util.Execute(util.ExecuteOptions{Io: appConfig.Io}, "git", "merge", mergeBase); err != nil {
		resolveMergeConflict(commit, appConfig)
	}
	applyCommitDiff(commit, appConfig)
	util.ExecuteOrDie(util.ExecuteOptions{Io: appConfig.Io}, "git", "commit", "-m", "Apply commit diff")
	gitutil.GitPushOrDie(util.ExecuteOptions{}, "push", "origin", branch+":"+branch)
	gitutil.GitSwitch(mainBranch)
	slog.Info(fmt.Sprint("Updated PR branch: ", branch))
}

func resolveMergeConflict(commit templates.GitLog, appConfig util.AppConfig) {
	defer func() {
		if r := recover(); r != nil {
			// nolint:errcheck
			util.Execute(util.ExecuteOptions{}, "git", "merge", "--abort")
			panic(r)
		}
	}()
	commitDiffFiles := strings.TrimSpace(util.ExecuteOrDie(util.ExecuteOptions{}, "git", "diff", "--name-only", commit.Commit+"~1", commit.Commit))
	commitDiffFileSet := make(map[string]bool)
	if commitDiffFiles != "" {
		for _, f := range strings.Split(commitDiffFiles, "\n") {
			commitDiffFileSet[f] = true
		}
	}
	conflictedFiles := strings.TrimSpace(util.ExecuteOrDie(util.ExecuteOptions{}, "git", "diff", "--name-only", "--diff-filter=U"))
	if conflictedFiles != "" {
		for _, f := range strings.Split(conflictedFiles, "\n") {
			if !commitDiffFileSet[f] {
				panic(fmt.Sprint("Merge conflict in a file that is not part of commit diff: ", f))
			}
		}
	}
	// Resolve the conflict by taking the mergeBase version; the content is transient
	// because applyCommitDiff overwrites these files with the commit's version next.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", "--theirs", ".")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")
	// GIT_EDITOR=true prevents merge --continue from opening an editor for the
	// merge commit message, which fails on CI where no editor is configured.
	continueOptions := util.ExecuteOptions{Io: appConfig.Io, EnvironmentVariables: []string{"GIT_EDITOR=true"}}
	util.ExecuteOrDie(continueOptions, "git", "merge", "--continue")
}

func applyCommitDiff(commit templates.GitLog, appConfig util.AppConfig) {
	util.ExecuteOrDie(util.ExecuteOptions{Io: appConfig.Io}, "git", "cherry-pick", "--no-commit", "--strategy-option=theirs", commit.Commit)
}
