package commands

import (
	"bufio"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createMigrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Migrates any work-in-progress branches to main. This prepares local git repository for first use by sd.",
		Long: `Migrates work-in-progress branches to main, preparing your local repository for stacked diff workflow.

This command is useful when first adopting sd in an existing repository with feature branches.
It will help you move commits from feature branches onto your main branch so they can be
managed as a stack.

Examples:
  sd migrate
  sd migrate --help`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			executeMigrate()
		},
	}
}

// executeMigrate implements the migration workflow for moving feature branches to main
func executeMigrate() {
	appConfig := util.GetAppConfig()
	// Step 1: Find all local branches where the current user has made commits
	slog.Debug("Step 1: Finding user branches...")
	userBranches := findUserBranches()
	slog.Debug(fmt.Sprintf("Step 1 complete: Found %d user branches", len(userBranches)))

	if len(userBranches) == 0 {
		util.Fprintln(appConfig.Io.Out, "No branches found with your commits")
		return
	}

	// Step 2: Display branches to user for selection
	slog.Debug("Step 2: Selecting branches to migrate...")
	selectedBranches := selectBranchesToMigrate(userBranches)
	slog.Debug(fmt.Sprintf("Step 2 complete: Selected %d branches", len(selectedBranches)))

	if len(selectedBranches) == 0 {
		slog.Debug("No branches selected for migration")
		appConfig.Exit(0)
	}

	slog.Debug(fmt.Sprintf("Selected %d branches for migration: %v", len(selectedBranches), selectedBranches))

	// Step 3: Find the most recent commit from origin/main
	mostRecentMainCommit := findMostRecentMainCommit(selectedBranches)
	slog.Debug(fmt.Sprintf("Most recent origin/main commit: %s", mostRecentMainCommit))

	// Step 4: Process each selected branch
	var results []migrationResult
	for _, branch := range slices.Backward(selectedBranches) {
		result := processBranch(branch, mostRecentMainCommit)
		results = append(results, result)
	}

	// Switch to main branch
	mainBranch := util.GetMainBranchOrDie()
	slog.Info(fmt.Sprintf("Switching to branch %s", mainBranch))
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", mainBranch)

	// Step 5: Report summary of migrated branches
	reportMigrationSummary(results)
}

// processBranch handles the migration of a single branch
func processBranch(branch string, baseCommit string) migrationResult {
	slog.Info(fmt.Sprintf("Processing branch: %s", branch))

	// Step 4f: Check if branch has a PR - skip if so
	mergedPR := util.GetMergedPR(branch)
	if mergedPR != nil {
		slog.Warn(fmt.Sprintf("Branch %s has merged PR #%s: %s - skipping migration", branch, mergedPR.Number, mergedPR.Title))
		return migrationResult{
			branchName: branch,
			success:    false,
			reason:     fmt.Sprintf("already merged (PR #%s)", mergedPR.Number),
			numCommits: 0,
		}
	}

	// Checkout and rebase the branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", branch)
	rebaseBranch(branch, baseCommit)

	// Get commits ahead of base
	commitsAhead := getCommitsAhead(baseCommit, "HEAD")
	slog.Debug(fmt.Sprintf("Found %d commits ahead of main for branch %s", len(commitsAhead), branch))
	for _, commit := range commitsAhead {
		slog.Debug(fmt.Sprintf("  - %s", commit))
	}

	if len(commitsAhead) == 0 {
		slog.Info(fmt.Sprintf("Branch %s has no commits ahead of main - skipping", branch))
		return migrationResult{
			branchName: branch,
			success:    false,
			reason:     "no commits ahead of main",
			numCommits: 0,
		}
	}

	// Step 4c: Check if branch has an unmerged PR
	pr := util.GetUnmergedPR(branch)
	if pr != nil {
		// Step 4d: Handle branch with unmerged PR
		// Branches with unmerged PRs are skipped because migrating them would create a new branch
		// name (from the template) that doesn't match the existing PR's branch.
		slog.Warn(fmt.Sprintf("Branch %s has an unmerged PR #%s: %s - skipping migration", branch, pr.Number, pr.Title))
		return migrationResult{
			branchName: branch,
			success:    false,
			reason:     fmt.Sprintf("Unmerged PR (#%s)", pr.Number),
			numCommits: 0,
		}
	} else {
		// Step 4e: Handle branch without PR
		return handleBranchWithoutPR(branch, baseCommit, commitsAhead)
	}
}

// rebaseBranch rebases the current branch onto baseCommit, handling errors appropriately
func rebaseBranch(branch string, baseCommit string) {
	appConfig := util.GetAppConfig()
	slog.Info("Rebasing branch " + branch + " onto most recent main commit " + baseCommit)
	util.RebaseAndSkipAllEmptyOrDie(
		util.ExecuteOptions{Io: util.StdIo{Out: appConfig.Io.Out, In: nil, Err: appConfig.Io.Err}},
		baseCommit)
	slog.Info(fmt.Sprintf("Successfully rebased branch: %s", branch))
}

// itemWithTimestamp associates a name with a timestamp for sorting purposes
type itemWithTimestamp struct {
	name      string
	timestamp int64
}

// migrationResult tracks the outcome of migrating a single branch
type migrationResult struct {
	branchName string
	success    bool
	reason     string // why it was skipped or failed
	numCommits int    // number of commits migrated
}

// parseTimestamp converts a git timestamp string to int64
func parseTimestamp(timestampStr string, context string) (int64, error) {
	timestamp, err := strconv.ParseInt(strings.TrimSpace(timestampStr), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse timestamp for %s: %w", context, err)
	}
	return timestamp, nil
}

// findUserBranches searches for all local branches where the current user has made commits,
// excluding the main branch, and sorts them by most recent commit timestamp (newest first).
func findUserBranches() []string {
	userEmail := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "config", "user.email")
	slog.Debug("Looking for branches with commits by: " + userEmail)

	mainBranch := util.GetMainBranchOrDie()

	// Get all local branches
	branchesOutput := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--format=%(refname:short)")
	if branchesOutput == "" {
		return []string{}
	}

	allBranches := strings.Split(branchesOutput, "\n")
	branchesWithTimestamps := filterBranchesWithUserCommits(allBranches, mainBranch, userEmail)

	// Sort by timestamp (most recent first)
	sort.Slice(branchesWithTimestamps, func(i, j int) bool {
		return branchesWithTimestamps[i].timestamp > branchesWithTimestamps[j].timestamp
	})

	// Extract branch names
	result := make([]string, len(branchesWithTimestamps))
	for i, b := range branchesWithTimestamps {
		result[i] = b.name
	}

	slog.Info(fmt.Sprintf("Found %d branches with your commits", len(result)))
	return result
}

// filterBranchesWithUserCommits returns branches (excluding mainBranch) where
// userEmail has commits, along with the timestamp of their most recent commit.
func filterBranchesWithUserCommits(allBranches []string, mainBranch string, userEmail string) []itemWithTimestamp {
	var branchesWithTimestamps []itemWithTimestamp

	for _, branch := range allBranches {
		branch = strings.TrimSpace(branch)
		if branch == "" || branch == mainBranch {
			continue
		}

		logOutput, err := util.Execute(util.ExecuteOptions{}, "git", "log", branch, "--author="+userEmail, "-1", "--format=%ct")
		if err != nil || strings.TrimSpace(logOutput) == "" {
			slog.Debug("Skipping branch " + branch + " (no commits by user)")
			continue
		}

		timestamp, err := parseTimestamp(logOutput, "branch "+branch)
		if err != nil {
			slog.Warn(err.Error())
			continue
		}

		slog.Debug(fmt.Sprintf("Found branch %s with user commits (timestamp: %s)", branch, strings.TrimSpace(logOutput)))
		branchesWithTimestamps = append(branchesWithTimestamps, itemWithTimestamp{
			name:      branch,
			timestamp: timestamp,
		})
	}

	return branchesWithTimestamps
}

// selectBranchesToMigrate displays an interactive selector for the user to choose which branches to migrate.
// Returns the selected branch names, or an empty array if user cancelled.
// Branches that already have their commits on main are displayed as disabled.
func selectBranchesToMigrate(branches []string) []string {
	appConfig := util.GetAppConfig()
	if !interactive.InteractiveEnabled() {
		slog.Warn("Interactive mode not available, migrating all branches")
		return branches
	}

	disabledBranches := computeDisabledBranches(branches)

	rowEnabled := func(row int) bool {
		if row < 0 || row >= len(branches) {
			return false
		}
		return !disabledBranches[row]
	}

	if len(disabledBranches) == len(branches) {
		slog.Info("All branches already exist on main - nothing to migrate")
		util.Fprintln(appConfig.Io.Out, "All branches have already been migrated to main")
		return []string{}
	}

	slog.Info("Starting interactive branch selection...")
	selectedBranches, err := interactive.GetBranchSelectionWithFilter(
		branches,
		"Select branches to migrate (use space to select/deselect, enter to confirm):",
		rowEnabled,
	)
	slog.Info("Interactive selection completed")
	if err != nil {
		slog.Warn("Failed to get branch selection: " + err.Error())
		return []string{}
	}

	return selectedBranches
}

// computeDisabledBranches returns a set of branch indices that already have their
// commits on main and should be disabled in the interactive selector.
func computeDisabledBranches(branches []string) map[int]bool {
	slog.Info("Fetching commits from main for branch filtering...")
	mainBranch := util.GetMainBranchOrDie()
	mainCommits := templates.GetNewCommits(mainBranch)
	slog.Info(fmt.Sprintf("Found %d commits on main for branch filtering", len(mainCommits)))

	branchesOnMain := make(map[string]bool)
	for _, commit := range mainCommits {
		if commit.Branch != "" {
			branchesOnMain[commit.Branch] = true
		}
	}

	slog.Info("Building branch filter...")
	disabledBranches := make(map[int]bool)
	for i, branch := range branches {
		if branchesOnMain[branch] {
			disabledBranches[i] = true
			slog.Info(fmt.Sprintf("Branch %s already exists on main - will be disabled", branch))
		}
	}
	return disabledBranches
}

// findMostRecentMainCommit finds the most recent commit from origin/main that is an ancestor
// of both the local main branch and all selected branches.
func findMostRecentMainCommit(branches []string) string {
	mainBranch := util.GetMainBranchOrDie()

	// Collect all branches to check (local main + selected branches)
	allBranches := append([]string{mainBranch}, branches...)

	var commits []itemWithTimestamp

	// For each branch, find its merge-base with origin/main
	for _, branch := range allBranches {
		mergeBase := util.FirstOriginMainCommit(branch)
		slog.Debug(fmt.Sprintf("Merge-base for %s with origin/main: %s", branch, mergeBase))

		// Get the timestamp of this commit
		timestampStr := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "log", "-1", "--format=%ct", mergeBase)
		timestamp, err := parseTimestamp(timestampStr, "commit "+mergeBase)
		if err != nil {
			slog.Warn(err.Error())
			continue
		}

		commits = append(commits, itemWithTimestamp{
			name:      mergeBase,
			timestamp: timestamp,
		})
	}

	if len(commits) == 0 {
		panic("No valid merge-base commits found")
	}

	// Find the most recent commit (highest timestamp)
	mostRecent := commits[0]
	for _, commit := range commits[1:] {
		if commit.timestamp > mostRecent.timestamp {
			mostRecent = commit
		}
	}

	return mostRecent.name
}

// getCommitsAhead returns a list of commit hashes between baseCommit and toRef (exclusive of baseCommit).
// The commits are returned in chronological order (oldest first).
func getCommitsAhead(baseCommit string, toRef string) []string {
	// Use git rev-list to get commits in reverse chronological order (newest first)
	// We'll reverse the list later to get oldest first
	output := util.ExecuteOrDieTrimmed(
		util.ExecuteOptions{},
		"git",
		"rev-list",
		"--reverse",
		"--abbrev-commit",
		baseCommit+".."+toRef,
	)

	if output == "" {
		return []string{}
	}

	commits := strings.Split(output, "\n")
	return commits
}

// promptForInput prompts the user for text input and returns their response.
func promptForInput(prompt string) string {
	appConfig := util.GetAppConfig()
	util.Fprint(appConfig.Io.Out, prompt+": ")
	scanner := bufio.NewScanner(appConfig.Io.In)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Sprintf("Failed to read input: %s", err.Error()))
	}
	return ""
}

// handleBranchWithoutPR handles step 4e: migration of a branch that doesn't have a PR.
func handleBranchWithoutPR(branch string, baseCommit string, commitsAhead []string) migrationResult {
	slog.Info(fmt.Sprintf("Branch %s does not have an unmerged PR", branch))

	// Prompt user for the eventual PR name
	prName := promptForInput("What should the PR be named when it is eventually created")
	if prName == "" {
		slog.Info(fmt.Sprintf("No PR name provided, skipping branch %s", branch))
		return migrationResult{
			branchName: branch,
			success:    false,
			reason:     "user cancelled (no PR name provided)",
			numCommits: 0,
		}
	}

	var commitPrefix string
	if len(commitsAhead) > 1 {
		// Prompt for prefix if there's more than one commit
		commitPrefix = promptForInput("Enter a short prefix to add to the other commits (or press enter to skip)")
	}

	// Rename the first commit to match the eventual PR title
	firstCommit := commitsAhead[0]
	renameCommitOnBranch(branch, firstCommit, prName)
	slog.Info(fmt.Sprintf("Renamed first commit to: %s", prName))

	// Rename remaining commits with prefix if provided
	if commitPrefix != "" && len(commitsAhead) > 1 {
		prefixRemainingCommits(branch, baseCommit, commitPrefix)
	}

	// Get the final list of commits in chronological order (oldest first)
	finalCommits := getCommitsAhead(baseCommit, branch)
	slog.Info(fmt.Sprintf("Final commits to cherry-pick (in order): %d commits", len(finalCommits)))
	for i, commit := range finalCommits {
		slog.Debug(fmt.Sprintf("  %d. %s: %s", i+1, commit, getCommitSubject(commit)))
	}

	// Cherry-pick each commit onto local main branch IN ORDER (oldest to newest)
	mainBranch := util.GetMainBranchOrDie()
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", mainBranch)

	slog.Info(fmt.Sprintf("Cherry-picking %d commits to %s (oldest to newest)", len(finalCommits), mainBranch))
	util.CherryPickAndSkipAllEmpty(finalCommits)

	return migrationResult{
		branchName: branch,
		success:    true,
		reason:     "migrated without PR",
		numCommits: len(finalCommits),
	}
}

// renameCommitOnBranch renames a commit on a branch.
// Returns the new commit hash after the rename.
// This works by checking out the commit, amending it, and then cherry-picking subsequent commits.
func renameCommitOnBranch(branch string, commitHash string, newSubject string) string {
	// Ensure we're on the correct branch
	currentBranch := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "--show-current")
	if currentBranch != branch {
		util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", branch)
	}

	// Get the current commit body (everything except the subject line)
	commitBody := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "-1", "--format=%b", commitHash)

	// Build the new message
	var newMessage string
	if commitBody != "" {
		newMessage = newSubject + "\n\n" + commitBody
	} else {
		newMessage = newSubject
	}

	// Get commits that come after this commit on the current branch
	branchHead := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", branch)
	commitsAfter := getCommitsAhead(commitHash, branchHead)

	// Checkout the commit and amend it
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", commitHash)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "--no-verify", "--amend", "-m", newMessage)

	// Cherry-pick subsequent commits and update branch pointer
	newHash := replayCommitsOnBranch(branch, commitsAfter)

	// Checkout the branch
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "checkout", branch)

	return newHash
}

// prefixRemainingCommits adds commitPrefix to the subject of all commits
// after the first one on the given branch.
func prefixRemainingCommits(branch string, baseCommit string, commitPrefix string) {
	currentCommits := getCommitsAhead(baseCommit, branch)

	for i := 1; i < len(currentCommits); i++ {
		commit := currentCommits[i]
		oldSubject := getCommitSubject(commit)
		newSubject := commitPrefix + " " + oldSubject
		renameCommitOnBranch(branch, commit, newSubject)
		slog.Info(fmt.Sprintf("Renamed commit to: %s", newSubject))

		// Get updated commit list after each rename since hashes change
		currentCommits = getCommitsAhead(baseCommit, branch)
	}
}

// replayCommitsOnBranch cherry-picks the given commits onto HEAD, updates the branch
// pointer, and returns the new HEAD hash. If commitsAfter is empty, just updates the
// branch pointer to the current HEAD.
func replayCommitsOnBranch(branch string, commitsAfter []string) string {
	newHash := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "HEAD")

	if len(commitsAfter) > 0 {
		for _, commit := range commitsAfter {
			util.ExecuteOrDie(util.ExecuteOptions{}, "git", "cherry-pick", commit)
		}
		util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "-f", branch, "HEAD")
		newHash = util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "HEAD")
	} else {
		util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "-f", branch, newHash)
	}

	return newHash
}

// getCommitSubject returns the subject line of a commit.
func getCommitSubject(commitHash string) string {
	return util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "-1", "--format=%s", commitHash)
}

// reportMigrationSummary reports the results of the migration process.
func reportMigrationSummary(results []migrationResult) {
	appConfig := util.GetAppConfig()
	util.Fprintln(appConfig.Io.Out, "")
	util.Fprintln(appConfig.Io.Out, "Migration Summary:")
	util.Fprintln(appConfig.Io.Out, strings.Repeat("=", 50))

	successCount := 0
	skippedCount := 0
	totalCommits := 0

	for _, result := range results {
		if result.success {
			successCount++
			totalCommits += result.numCommits
			util.Fprintln(appConfig.Io.Out, fmt.Sprintf("✓ %s: migrated %d commit(s) - %s",
				result.branchName, result.numCommits, result.reason))
		} else {
			skippedCount++
			util.Fprintln(appConfig.Io.Out, fmt.Sprintf("⊘ %s: skipped - %s",
				result.branchName, result.reason))
		}
	}

	util.Fprintln(appConfig.Io.Out, strings.Repeat("=", 50))
	util.Fprintln(appConfig.Io.Out, fmt.Sprintf("Total: %d migrated, %d skipped, %d commits moved to main",
		successCount, skippedCount, totalCommits))
}
