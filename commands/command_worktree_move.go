package commands

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
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
		Annotations: map[string]string{
			checkRepoAnnotation: "true",
		},
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
			if wt.Path == worktreeFlag || wt.BranchOrCommit == worktreeFlag || filepath.Base(wt.Path) == worktreeFlag {
				return wt.Path
			}
		}
		panic("Worktree not found: " + worktreeFlag)
	}
	// Prompt user to select a worktree interactively.
	if !interactive.InteractiveEnabled() {
		panic("Not in a secondary worktree and cannot ask interactively because not a terminal. Use --worktree to specify the source worktree.")
	}
	options := make([]interactive.WorktreeOption, len(worktrees))
	for i, wt := range worktrees {
		options[i] = interactive.WorktreeOption{Branch: wt.BranchOrCommit, Path: wt.Path}
	}
	selectedIndex, err := interactive.GetWorktreeSelection(options, "Which worktree do you want to move commits from?")
	if err != nil {
		panic(err.Error())
	}
	if selectedIndex < 0 {
		util.GetAppConfig().Exit(0)
	}
	return worktrees[selectedIndex].Path
}

func worktreeMove(args []string, indicatorTypeString *string, worktreeFlag string) {
	secondaryPath := resolveSecondaryWorktree(worktreeFlag)
	mainPath := gitutil.GetMainWorktreePath()
	mainBranch := gitutil.GetRemoteMainBranchOrDie()
	// Build set of branches already on main so they are disabled in the selector.
	mainCommits := templates.GetNewCommits("HEAD", mainPath)
	disabledBranches := make(map[string]bool, len(mainCommits))
	for _, c := range mainCommits {
		disabledBranches[c.Branch] = true
	}
	selectCommitOptions := interactive.CommitSelectionOptions{
		Prompt:           "What commits do you want to move to the main worktree?",
		CommitType:       interactive.CommitTypeBoth,
		MultiSelect:      true,
		GitDir:           secondaryPath,
		DisabledBranches: disabledBranches,
	}
	selectedCommits := getTargetCommits(args, indicatorTypeString, selectCommitOptions)
	slog.Info(fmt.Sprint("Cherry-picking ", len(selectedCommits), " commit(s) onto ", mainBranch, " in main worktree"))
	commits := util.MapSlice(selectedCommits, func(commit templates.GitLog) string {
		return commit.Commit
	})
	cherryPickWithRecovery(mainPath, commits, cherryPickRecoveryOptions{
		OnRollback: func() {
			util.ExecuteOrDie(util.ExecuteOptions{}, "git", gitutil.PrependGitDir(mainPath, "cherry-pick", "--abort"))
		},
		OnContinueManually: func() {
			if err := os.Chdir(mainPath); err != nil {
				panic(err)
			}
			slog.Info(fmt.Sprint("Changed directory to main worktree: ", mainPath))
		},
	})
	slog.Info(fmt.Sprint("Successfully cherry-picked ", len(selectedCommits), " commit(s) onto ", mainBranch))
}
