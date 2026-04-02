package commands

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createWorktreeMoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worktree-move [commitIndicator...]",
		Short: "Cherry-pick commits from secondary worktree to main worktree",
		Long: "Cherry-picks selected commits from a secondary worktree\n" +
			"to the main worktree. Useful for when you want to build from one\n" +
			"directory with all of your changes.\n" +
			"\n" +
			"Can be run from a secondary worktree or the main worktree.\n" +
			"When run from the main worktree, use --worktree to specify\n" +
			"the source worktree, or select one interactively.",
		Args: cobra.ArbitraryArgs,
	}
	indicatorTypeString := addIndicatorFlag(cmd)
	var worktreeFlag string
	cmd.Flags().StringVarP(&worktreeFlag, "worktree", "w", "", "Path or branch name of the secondary worktree to move commits from")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		worktreeMove(args, indicatorTypeString, worktreeFlag)
	}
	return cmd
}

// resolveSecondaryWorktree determines which secondary worktree to use.
// If already in a secondary worktree, returns empty string (use current directory).
// Otherwise returns the path to the selected secondary worktree.
func resolveSecondaryWorktree(worktreeFlag string) string {
	if gitutil.IsSecondaryWorktree() {
		return ""
	}
	worktrees := gitutil.GetSecondaryWorktrees()
	if len(worktrees) == 0 {
		panic("No secondary worktrees found")
	}
	if worktreeFlag != "" {
		for _, wt := range worktrees {
			if wt.Path == worktreeFlag || wt.Branch == worktreeFlag || filepath.Base(wt.Path) == worktreeFlag {
				return wt.Path
			}
		}
		panic("Worktree not found: " + worktreeFlag)
	}
	// Prompt user to select a worktree interactively.
	if !interactive.InteractiveEnabled() {
		panic("Not in a secondary worktree and cannot ask interactively because not a terminal. Use --worktree to specify the source worktree.")
	}
	branches := make([]string, len(worktrees))
	for i, wt := range worktrees {
		branches[i] = wt.Branch
	}
	selected, err := interactive.GetBranchSelection(branches, "Which worktree do you want to move commits from?")
	if err != nil {
		panic(err.Error())
	}
	if len(selected) == 0 {
		util.GetAppConfig().Exit(0)
	}
	// Find the worktree path for the selected branch.
	for _, wt := range worktrees {
		if wt.Branch == selected[0] {
			return wt.Path
		}
	}
	panic("Selected worktree not found")
}

func worktreeMove(args []string, indicatorTypeString *string, worktreeFlag string) {
	secondaryPath := resolveSecondaryWorktree(worktreeFlag)
	selectCommitOptions := interactive.CommitSelectionOptions{
		Prompt:      "What commits do you want to move to the main worktree?",
		CommitType:  interactive.CommitTypeBoth,
		MultiSelect: true,
	}
	// If we need to operate from a different worktree, chdir there for commit selection.
	if secondaryPath != "" {
		originalDir, err := os.Getwd()
		if err != nil {
			panic("Failed to get current directory: " + err.Error())
		}
		if err := os.Chdir(secondaryPath); err != nil {
			panic("Failed to change to worktree directory: " + err.Error())
		}
		defer func() {
			_ = os.Chdir(originalDir)
		}()
	}
	selectedCommits := getTargetCommits(args, indicatorTypeString, selectCommitOptions)
	mainPath := gitutil.GetMainWorktreePath()
	mainBranch := gitutil.GetRemoteMainBranchOrDie()
	savedHead := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "-C", mainPath, "log", "-n", "1", "--pretty=format:%H")
	slog.Info(fmt.Sprint("Cherry-picking ", len(selectedCommits), " commit(s) onto ", mainBranch, " in main worktree"))
	cherryPickArgs := make([]string, 0, 3+len(selectedCommits))
	cherryPickArgs = append(cherryPickArgs, "-C", mainPath, "cherry-pick")
	for _, commit := range selectedCommits {
		cherryPickArgs = append(cherryPickArgs, commit.Commit)
	}
	out, cherryPickErr := util.Execute(util.ExecuteOptions{}, "git", cherryPickArgs...)
	if cherryPickErr != nil {
		slog.Error(fmt.Sprint("Cherry-pick failed: ", out))
		if interactive.Confirm("Cherry-pick failed. Rollback changes in main worktree?", true) {
			_, _ = util.Execute(util.ExecuteOptions{}, "git", "-C", mainPath, "cherry-pick", "--abort")
			util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-C", mainPath, "reset", "--hard", savedHead)
			slog.Info("Rolled back " + mainBranch + " to " + savedHead)
		} else {
			slog.Info("Leaving main worktree as-is. Resolve conflicts manually in " + mainPath)
		}
		return
	}
	slog.Info(fmt.Sprint("Successfully cherry-picked ", len(selectedCommits), " commit(s) onto ", mainBranch))
}
