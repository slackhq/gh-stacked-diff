package commands

import (
	"flag"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/templates"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

const (
	onCherryPickErrorPrompt   = "prompt"
	onCherryPickErrorRollback = "rollback"
	onCherryPickErrorExit     = "exit"
)

func createReplaceCommitCommand() Command {
	flagSet := flag.NewFlagSet("replace-commit", flag.ContinueOnError)
	indicatorTypeString := addIndicatorFlag(flagSet)
	onCherryPickError := flagSet.String("on-cherry-pick-error", onCherryPickErrorPrompt,
		"Action when cherry-pick fails: prompt, rollback, or exit")
	return Command{
		FlagSet: flagSet,
		Summary: "Replaces a commit on " + util.GetMainBranchForHelp() + " branch with its associated branch",
		Description: "Replaces a commit on " + util.GetMainBranchForHelp() + " branch with the squashed contents of its\n" +
			"associated branch.\n" +
			"\n" +
			"This is useful when you make changes within a branch, for example to\n" +
			"fix a problem found on CI, and want to bring the changes over to your\n" +
			"local " + util.GetMainBranchForHelp() + " branch.",
		Usage: "sd " + flagSet.Name() + " [flags] <commitIndicator>",
		OnSelected: func(appConfig util.AppConfig, command Command) {
			if flagSet.NArg() > 1 {
				commandError(appConfig, flagSet, "too many arguments", command.Usage)
			}
			selectCommitOptions := interactive.CommitSelectionOptions{
				Prompt:      "What commit do you want to replace with the contents of its associated branch?",
				CommitType:  interactive.CommitTypePr,
				MultiSelect: false,
			}
			util.RequireMainBranch()
			targetCommit := getTargetCommits(appConfig, command, []string{flagSet.Arg(0)}, indicatorTypeString, selectCommitOptions)
			replaceCommit(appConfig, *onCherryPickError, targetCommit[0])
		}}
}

// Replaces a commit on main branch with its associated branch.
func replaceCommit(appConfig util.AppConfig, onCherryPickError string, targetCommit templates.GitLog) {
	templates.RequireCommitOnMain(targetCommit.Commit)
	util.WithStashAndRollback("replace-commit "+targetCommit.Commit+" "+targetCommit.Subject, func(rollbackManager *util.GitRollbackManager) {
		replaceCommitOfBranchInfo(appConfig, rollbackManager, onCherryPickError, targetCommit)
		rollbackManager.Clear()
	})
}

// Replaces commit gitLog.Commit with the contents of branch gitLog.Branch.
func replaceCommitOfBranchInfo(appConfig util.AppConfig, rollbackManager *util.GitRollbackManager, onCherryPickError string, gitLog templates.GitLog) {
	rollbackCommit := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "log", "-n", "1", "--pretty=format:%H")
	commitsAfter := strings.Fields(util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "--no-pager", "log", gitLog.Commit+"..HEAD", "--pretty=format:%h"))
	slices.Reverse(commitsAfter)
	commitToDiffFrom := util.FirstOriginMainCommit(gitLog.Branch)
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
			util.CherryPickAndSkipAllEmpty(commitsAfter)
			return nil
		}()
		if cherryPickErr != nil {
			util.Fprintln(appConfig.Io.Out, fmt.Sprint("Cherry-pick failed: ", cherryPickErr))
			shouldRollback := onCherryPickError == onCherryPickErrorRollback ||
				(onCherryPickError == onCherryPickErrorPrompt && interactive.Confirm(appConfig, "Rollback all changes?", true))
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
