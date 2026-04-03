package commands

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/templates"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

const (
	onCherryPickErrorPrompt   = "prompt"
	onCherryPickErrorRollback = "rollback"
	onCherryPickErrorExit     = "exit"
)

func createReplaceCommitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replace-commit [commitIndicator]",
		Short: "Replaces a commit on " + gitutil.GetMainBranchForHelp() + " branch with its associated branch",
		Long: "Replaces a commit on " + gitutil.GetMainBranchForHelp() + " branch with the squashed contents of its\n" +
			"associated branch.\n" +
			"\n" +
			"This is useful when you make changes within a branch, for example to\n" +
			"fix a problem found on CI, and want to bring the changes over to your\n" +
			"local " + gitutil.GetMainBranchForHelp() + " branch.",
		Args: cobra.MaximumNArgs(1),
	}
	indicatorTypeString := addIndicatorFlag(cmd)
	onCherryPickError := cmd.Flags().String("on-cherry-pick-error", onCherryPickErrorPrompt,
		"Action when cherry-pick fails: prompt, rollback, or exit")
	_ = cmd.RegisterFlagCompletionFunc("on-cherry-pick-error", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{onCherryPickErrorPrompt, onCherryPickErrorRollback, onCherryPickErrorExit}, cobra.ShellCompDirectiveDefault
	})
	cmd.Run = func(cmd *cobra.Command, args []string) {
		selectCommitOptions := interactive.CommitSelectionOptions{
			Prompt:      "What commit do you want to replace with the contents of its associated branch?",
			CommitType:  interactive.CommitTypePr,
			MultiSelect: false,
		}
		gitutil.RequireMainBranch()
		targetCommit := getTargetCommits(args, indicatorTypeString, selectCommitOptions)
		replaceCommit(*onCherryPickError, targetCommit[0])
	}
	return cmd
}

// Replaces a commit on main branch with its associated branch.
func replaceCommit(onCherryPickError string, targetCommit templates.GitLog) {
	templates.RequireCommitOnMain(targetCommit.Commit)
	gitutil.WithStashAndRollback("replace-commit "+targetCommit.Commit+" "+targetCommit.Subject, func(rollbackManager *gitutil.GitRollbackManager) {
		replaceCommitOfBranchInfo(rollbackManager, onCherryPickError, targetCommit)
		rollbackManager.Clear()
	})
}

// Replaces commit gitLog.Commit with the contents of branch gitLog.Branch.
func replaceCommitOfBranchInfo(rollbackManager *gitutil.GitRollbackManager, onCherryPickError string, gitLog templates.GitLog) {
	appConfig := util.GetAppConfig()
	rollbackCommit := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "-n", "1", "--pretty=format:%H")
	commitsAfter := strings.Fields(util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "--no-pager", "log", gitLog.Commit+"..HEAD", "--pretty=format:%h"))
	slices.Reverse(commitsAfter)
	commitToDiffFrom := gitutil.FirstOriginMainCommit(gitLog.Branch)
	slog.Info("Resetting to " + gitLog.Commit + "~1")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", gitLog.Commit+"~1")
	slog.Info("Adding diff from commits " + gitLog.Branch)
	diff := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "diff", "--binary", commitToDiffFrom, gitLog.Branch)
	util.ExecuteOrDie(
		util.ExecuteOptions{Io: util.StdIo{In: strings.NewReader(diff), Out: nil, Err: nil}},
		"git", "apply",
	)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")
	commitSummary := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "--no-pager", "show", "--no-patch", "--format=%s", gitLog.Commit)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "--no-verify", "-m", strings.TrimSpace(commitSummary))
	if len(commitsAfter) != 0 {
		slog.Info(fmt.Sprint("Cherry picking commits back on top ", commitsAfter))
		cherryPickErr := func() (r any) {
			defer func() { r = recover() }()
			gitutil.CherryPickAndSkipAllEmpty("", commitsAfter)
			return nil
		}()
		if cherryPickErr != nil {
			util.Fprintln(appConfig.Io.Out, fmt.Sprint("Cherry-pick failed: ", cherryPickErr))
			shouldRollback := onCherryPickError == onCherryPickErrorRollback ||
				(onCherryPickError == onCherryPickErrorPrompt && interactive.Confirm("Rollback all changes?", true))
			if shouldRollback {
				panic(cherryPickErr)
			}
			util.Fprintln(appConfig.Io.Out, "To resolve manually:")
			util.Fprintln(appConfig.Io.Out, "  1. Fix the conflicts")
			util.Fprintln(appConfig.Io.Out, "  2. git add <resolved files>")
			util.Fprintln(appConfig.Io.Out, "  3. git cherry-pick --continue")
			util.Fprintln(appConfig.Io.Out, "  Repeat steps 1-3 until cherry-pick is complete.")
			util.Fprintln(appConfig.Io.Out, "  4. git stash pop (if you had stashed changes)")
			util.Fprintln(appConfig.Io.Out, "To abort:")
			util.Fprintln(appConfig.Io.Out, "  1. git cherry-pick --abort")
			util.Fprintln(appConfig.Io.Out, "  2. git reset --hard "+rollbackCommit)
			util.Fprintln(appConfig.Io.Out, "  3. git stash pop (if you had stashed changes)")
			rollbackManager.SkipRestore()
			appConfig.Exit(0)
		}
	}
}
