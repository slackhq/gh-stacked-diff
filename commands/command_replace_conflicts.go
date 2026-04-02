package commands

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/templates"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createReplaceConflictsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "replace-conflicts",
		Short: "For failed rebase: replace changes with its associated branch",
		Long: "During a rebase that failed because of merge conflicts, replace the\n" +
			"current uncommitted changes (merge conflicts), with the contents\n" +
			"(diff between origin/" + gitutil.GetMainBranchForHelp() + " and HEAD) of its associated branch.",
		Args: cobra.NoArgs,
	}
	confirmed := cmd.Flags().BoolP("confirm", "y", false, "Whether to automatically confirm to do this rather than ask for y/n input")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		replaceConflicts(*confirmed)
	}
	return cmd
}

// For failed rebase: replace changes with its associated branch.
func replaceConflicts(confirmed bool) {
	commitWithConflicts := getCommitWithConflicts()
	gitLog := templates.GetBranchInfo(commitWithConflicts, templates.IndicatorTypeCommit)
	checkConfirmed(confirmed)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", "HEAD")
	slog.Info(fmt.Sprint("Replacing changes (merge conflicts) for failed rebase of commit ", commitWithConflicts, ", with changes from associated branch, ", gitLog.Branch))
	diff := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "diff", "--binary", "origin/"+gitutil.GetRemoteMainBranchOrDie(), gitLog.Branch)
	util.ExecuteOrDie(util.ExecuteOptions{Io: util.StdIo{In: strings.NewReader(diff), Out: nil, Err: nil}},
		"git", "apply",
	)
	slog.Info("Adding changes and continuing rebase")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")
	continueOptions := util.ExecuteOptions{EnvironmentVariables: []string{"GIT_EDITOR=true"}}
	// Note: --continue cannot be used with --no-verify.
	util.ExecuteOrDie(continueOptions, "git", "rebase", "--continue")
}

func getCommitWithConflicts() string {
	statusLines := strings.Split(util.ExecuteOrDie(util.ExecuteOptions{}, "git", "status"), "\n")
	lastCommandDoneLine := -1
	inLast := false
	for i, line := range statusLines {
		if strings.HasPrefix(line, "Last ") {
			// find last pick line
			inLast = true
		} else if inLast {
			if strings.HasPrefix(line, "   ") {
				lastCommandDoneLine = i
			} else {
				break
			}
		}
	}
	if lastCommandDoneLine == -1 {
		panic("Cannot determine which commit is being rebased with because \"git status\" does not have a \"Last commands done\" section. To use this command you must be in the middle of a rebase")
	}
	// Return the 2nd field, from a string such as "pick f52e867 next1"
	return strings.Fields(statusLines[lastCommandDoneLine])[1]
}

func checkConfirmed(confirmed bool) {
	if confirmed {
		return
	}
	interactive.ConfirmOrDie("This will clear any uncommitted changes, are you sure (y/n)?", false)
}
