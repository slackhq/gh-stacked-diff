package commands

import (
	"fmt"
	"log/slog"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createWorktreeMoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worktree-move [commitIndicator...]",
		Short: "Cherry-pick commits from secondary worktree to main worktree",
		Long: "Cherry-picks selected commits from the current secondary worktree\n" +
			"to the main worktree. Useful for when you want to build from one\n" +
			"directory with all of your changes.\n" +
			"\n" +
			"Must be run from a secondary worktree.",
		Args: cobra.ArbitraryArgs,
	}
	indicatorTypeString := addIndicatorFlag(cmd)
	cmd.Run = func(cmd *cobra.Command, args []string) {
		worktreeMove(args, indicatorTypeString)
	}
	return cmd
}

func worktreeMove(args []string, indicatorTypeString *string) {
	if !gitutil.IsSecondaryWorktree() {
		panic("Must be run from a secondary worktree")
	}
	selectCommitOptions := interactive.CommitSelectionOptions{
		Prompt:      "What commits do you want to move to the main worktree?",
		CommitType:  interactive.CommitTypeBoth,
		MultiSelect: true,
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
