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

func createRebaseMainCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rebase-main",
		Short: "Bring your main branch up to date with remote",
		Long: "Rebase with origin/" + gitutil.GetMainBranchForHelp() + ", dropping any commits who's associated\n" +
			"branches have been merged or closed.\n" +
			"\n" +
			"Commits from merged PRs are automatically dropped. For commits from closed\n" +
			"(not merged) PRs, you will be prompted to confirm before dropping them.\n" +
			"\n" +
			"This avoids having to manually call \"git reset --hard head\" whenever\n" +
			"you have merge conflicts with a commit that has already been merged\n" +
			"but has slight variation with local main because, for example, a\n" +
			"change was made with the Github Web UI.\n" +
			"\n" +
			"Note: git hooks (pre-commit, etc.) are bypassed during the rebase. This is\n" +
			"safe because commits must be cherry-picked onto a PR branch via \"sd new\" or\n" +
			"\"sd update\" before they can be pushed, which runs hooks normally.",
		Args: cobra.NoArgs,
		Annotations: map[string]string{
			checkRepoAnnotation: "true",
		},
		Run: func(cmd *cobra.Command, args []string) {
			rebaseMain()
		},
	}
}

// Bring local main branch up to date with remote
func rebaseMain() {
	appConfig := util.GetAppConfig()
	gitutil.RequireMainBranch()
	shouldPopStash := gitutil.Stash("rebase-main")

	slog.Info("Fetching...")
	util.ExecuteOrDie(util.ExecuteOptions{Io: appConfig.Io}, "git", "fetch")
	slog.Info("Getting merged branches from Github...")
	mergedBranches := getMergedBranches()

	slog.Info("Getting closed branches from Github...")
	closedBranches := getClosedBranches()

	localLogs := templates.GetNewCommits("HEAD", "")
	mergedCommits := getDropCommits(localLogs, mergedBranches)
	slog.Debug(fmt.Sprint("mergedBranches ", mergedBranches, " mergedCommits ", mergedCommits))

	// Check for commits from closed PRs
	closedCommits := getDropCommits(localLogs, closedBranches)
	slog.Debug(fmt.Sprint("closedBranches ", closedBranches, " closedCommits ", closedCommits))

	var confirmedClosedCommits []templates.GitLog
	if len(closedCommits) > 0 {
		slog.Info(fmt.Sprint("Found ", len(closedCommits), " commits from closed (not merged) PRs:"))
		for _, closedCommit := range closedCommits {
			slog.Info(fmt.Sprint("  - ", closedCommit.Branch, ": ", closedCommit.Subject))
		}
		if interactive.Confirm("Do you want to drop these closed PR commits and delete their local branches?", false) {
			confirmedClosedCommits = closedCommits
		} else {
			slog.Info("Skipping closed PR commits")
		}
	}

	// Combine all commits to drop for rebase
	allDropCommits := append(mergedCommits, confirmedClosedCommits...)

	slog.Info("Rebasing...")
	var rebaseError error
	if len(allDropCommits) > 0 {
		commitHashes := util.MapSlice(allDropCommits, func(gitLog templates.GitLog) string {
			return gitLog.Commit
		})
		environmentVariables := []string{
			sequenceEditorEnvVar("sequence-editor-drop-already-merged", commitHashes...),
		}
		options := util.ExecuteOptions{
			EnvironmentVariables: environmentVariables,
			Io:                   appConfig.Io,
		}
		_, rebaseError = gitutil.RebaseAndSkipAllEmpty(options, "-i", "origin/"+gitutil.GetRemoteMainBranchOrDie())
		// Delete branches even if rebase failed. This is safe because these are
		// branches for PRs already merged/closed on GitHub — they are not the local
		// working branch. Cleaning them up regardless avoids stale branch accumulation.
		slog.Info("Deleting branches...")
		// Delete merged branches (including remote)
		deleteBranches(appConfig.Io, mergedCommits, true)
		// Delete closed PR branches (local only)
		deleteBranches(appConfig.Io, confirmedClosedCommits, false)
	} else {
		options := util.ExecuteOptions{Io: appConfig.Io}
		_, rebaseError = gitutil.RebaseAndSkipAllEmpty(options, "origin/"+gitutil.GetRemoteMainBranchOrDie())
	}
	if rebaseError != nil {
		if shouldPopStash {
			slog.Info("Your changes are still stashed. Run `git stash pop` after resolving the rebase.")
		}
		panic("Rebase failed, check output ^^ for details. Continue rebase manually.")
	}
	// Only pop stash on success — popping onto merge conflicts would be a problem.
	gitutil.PopStash(shouldPopStash)
}

// getMergedBranches returns branches from merged PRs that were merged AFTER our branch diverged.
//
// We only include branches where the merge commit is NOT an ancestor of HEAD, meaning the PR
// was merged after we created our branch. We should drop our local version since it's already
// incorporated in origin/main (possibly with slight differences due to squashing or web UI edits).
// This avoids rebase conflicts with commits that are effectively already merged.
func getMergedBranches() []string {
	return getBranchesByPRState(true)
}

// getClosedBranches returns branches from all closed (not merged) PRs.
//
// We skip ancestry checking because gh-stacked-diff creates PRs by cherry-picking commits
// to new branches, resulting in different SHAs for the same logical commit. Instead, we
// return all closed PR branch names and rely on branch name matching in getDropCommits()
// to identify which closed PRs correspond to local commits. This allows users to clean up
// abandoned work by dropping commits from closed PRs that match their local branch names.
func getClosedBranches() []string {
	return getBranchesByPRState(false)
}

// getBranchesByPRState fetches branches from GitHub PRs (merged or closed)
// and optionally filters them based on git ancestry relative to the current HEAD.
//
// Parameters:
//   - mergedState: if true, query merged PRs; if false, query closed PRs
//
// Behavior:
//
//   - Merged PRs: Returns branches that were merged AFTER our current branch diverged.
//     Only includes branches where the merge commit is NOT an ancestor of HEAD, meaning
//     it was merged after we branched off. We should drop our local version since it's
//     already in origin/main (possibly with slight differences due to squashing or web UI edits).
//
//   - Closed PRs: Returns all closed PR branches without ancestry filtering.
//     gh-stacked-diff creates PRs by cherry-picking commits, which generates different
//     commit SHAs on the PR branch vs main. Ancestry checks would fail even for commits
//     that are logically the same. Instead, we rely on branch name matching in getDropCommits()
//     to identify which closed PRs correspond to local commits.
func getBranchesByPRState(mergedState bool) []string {
	var branchesRaw string
	if mergedState {
		branchesRaw = util.ExecuteOrDie(util.ExecuteOptions{Retries: gitutil.GhRetries},
			"gh", "pr", "list", "--author", "@me", "--state", "merged", "--base", gitutil.GetRemoteMainBranchOrDie(),
			"--json", "headRefName,mergeCommit", "--jq", ".[ ] | .headRefName + \" \" +  .mergeCommit.oid")
	} else {
		branchesRaw = util.ExecuteOrDie(util.ExecuteOptions{Retries: gitutil.GhRetries},
			"gh", "pr", "list", "--author", "@me", "--state", "closed", "--search", "is:unmerged", "--base", gitutil.GetRemoteMainBranchOrDie(),
			"--json", "headRefName,headRefOid", "--jq", ".[ ] | .headRefName + \" \" + .headRefOid")
	}
	branchesRawLines := strings.Split(strings.TrimSpace(branchesRaw), "\n")
	branches := make([]string, 0, len(branchesRawLines))

	for _, branchRawLine := range branchesRawLines {
		fields := strings.Fields(branchRawLine)
		if len(fields) != 2 {
			continue
		}
		// Validate refs from GitHub API before passing to git commands.
		util.RequireGitRef(fields[0])
		util.RequireHexString(fields[1])

		// For closed PRs, skip ancestry check and include all branches
		if !mergedState {
			branches = append(branches, fields[0])
			continue
		}

		// For merged PRs, check if merge commit is an ancestor
		// git merge-base --is-ancestor <commit> HEAD checks if <commit> is reachable
		// from HEAD through parent relationships. Returns 0 (success) if true, 1 if false.
		_, mergeBaseErr := util.Execute(util.ExecuteOptions{}, "git", "merge-base", "--is-ancestor", fields[1], "HEAD")
		isAncestor := mergeBaseErr == nil

		// Only include merged branches where commit is NOT an ancestor (merged after we branched)
		if !isAncestor {
			branches = append(branches, fields[0])
		}
	}
	return branches
}

func getDropCommits(localLogs []templates.GitLog, branches []string) []templates.GitLog {
	// Look for matching branches between localLogs and the provided branch list
	var dropCommits []templates.GitLog
	for _, localLog := range localLogs {
		if slices.Contains(branches, localLog.Branch) {
			dropCommits = append(dropCommits, localLog)
		}
	}
	// Verify that there is only one local commit with that hash
	checkUniqueBranches(dropCommits)
	return dropCommits
}

// panics if there are duplicate branches in dropCommits.
func checkUniqueBranches(dropCommits []templates.GitLog) {
	branchToCommit := make(map[string]string)
	for _, dropCommit := range dropCommits {
		if otherCommit, ok := branchToCommit[dropCommit.Branch]; ok {
			panic(fmt.Sprint("Multiple commits, (", dropCommit.Commit, ", ", otherCommit, "), have the same branch:\n",
				dropCommit.Branch, "\n",
				"Ensure that all the commits in the diff stack have unique commit summaries."))
		}
		branchToCommit[dropCommit.Branch] = dropCommit.Commit
	}
}

func deleteBranches(stdIo util.StdIo, dropCommits []templates.GitLog, deleteRemote bool) {
	for _, dropCommit := range dropCommits {
		localHash := gitutil.GetBranchLatestCommit(dropCommit.Branch)
		if localHash != "" {
			// nolint:errcheck
			util.Execute(util.ExecuteOptions{Io: stdIo}, "git", "branch", "-D", dropCommit.Branch)
			// Only delete remote branch if requested and if it is on the same commit to avoid
			// accidentally deleting a branch that is not merged.
			if deleteRemote && localHash == gitutil.GetBranchLatestCommit("origin/"+dropCommit.Branch) {
				// nolint:errcheck
				util.Execute(util.ExecuteOptions{Io: stdIo}, "git", "push", "--delete", "origin", dropCommit.Branch)
			}
		}
	}
}
