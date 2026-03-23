package commands

import (
	"slices"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createLogCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Displays git log of your changes",
		Long: "Displays summary of the git commits on current branch that are not\n" +
			"in the remote branch.\n" +
			"\n" +
			"Useful to view list indexes, or copy commit hashes, to use for the\n" +
			"commitIndicator required by other commands.\n" +
			"\n" +
			"A " + color.GreenString("✅") + " means that there is a PR associated with the commit (actually it\n" +
			"means there is a branch, but having a branch means there is a PR when\n" +
			"using this workflow). If there is more than one commit on the\n" +
			"associated branch, those commits are also listed (indented under the\n" +
			"their associated commit summary).",
		Args: cobra.NoArgs,
		Annotations: map[string]string{
			"defaultLogLevel": "error",
		},
	}
	status := cmd.Flags().BoolP("status", "s", false,
		"Show PR status including checks, approvals, and state.\n"+
			"Only supported on the main branch.")
	poll := cmd.Flags().BoolP("poll", "p", false,
		"Keep polling for status updates. Requires --status.\n"+
			"Press Esc or Ctrl+C to exit.")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		if *poll && !*status {
			panic("--poll requires --status")
		}
		if *status {
			printGitLogWithStatus(cmd, *poll)
		} else {
			printGitLog()
		}
	}
	return cmd
}

func printGitLogWithStatus(cmd *cobra.Command, poll bool) {
	if util.GetCurrentBranchName() != util.GetMainBranchOrDie() {
		panic("--status is only supported on the main branch")
	}
	logs, checkedBranches := getLogsAndBranches()
	var pollInterval time.Duration
	if poll {
		pollInterval = getUserConfig(cmd).PollInterval
	}
	interactive.ShowLogStatus(logs, checkedBranches, pollInterval, getLogsAndBranches)
}

// Prints changes in the current branch compared to the main branch to out.
func printGitLog() {
	stdIo := util.GetAppConfig().Io
	if util.GetCurrentBranchName() != util.GetMainBranchOrDie() {
		gitArgs := []string{"--no-pager", "log", "--pretty=oneline", "--abbrev-commit"}
		if util.RemoteHasBranch(util.GetMainBranchOrDie()) {
			gitArgs = append(gitArgs, "origin/"+util.GetMainBranchOrDie()+"..HEAD")
		}
		gitArgs = append(gitArgs, "--color=always")
		util.ExecuteOrDie(util.ExecuteOptions{Io: stdIo}, "git", gitArgs...)
		return
	}
	logs, checkedBranches := getLogsAndBranches()
	for i, log := range logs {
		numberPrefix := interactive.GetLogNumberPrefix(i, len(logs))
		if slices.Contains(checkedBranches, log.Branch) {
			// Use color for ✅ otherwise in Git Bash on Windows it will appear as black and white.
			util.Fprint(stdIo.Out, numberPrefix+color.GreenString("✅ "))
		} else {
			util.Fprint(stdIo.Out, numberPrefix+"   ")
		}
		util.Fprintln(stdIo.Out, color.YellowString(log.Commit)+" "+log.Subject)
		// find first commit that is not in main branch
		if slices.Contains(checkedBranches, log.Branch) {
			branchCommits := templates.GetNewCommits(log.Branch)
			if len(branchCommits) > 1 {
				// Reverse the commits so that they are ordered from the earliest to more recent.
				slices.Reverse(branchCommits)
				// Remove the earliest commit as it is equal to newCommit
				branchCommits = branchCommits[1:]
				// Only include the three most recent commits, if there are more than three
				// commits use a "hiding xx" message
				padding := strings.Repeat(" ", len(numberPrefix))
				if len(branchCommits) > 3 {
					hidingColor := color.New(color.Italic).AddRGB(88, 88, 88) // Dark gray
					hidingMessage := hidingColor.Sprint("   - [hiding ", (len(branchCommits) - 2), " previous...]")
					util.Fprintln(stdIo.Out, padding+hidingMessage)
					branchCommits = branchCommits[len(branchCommits)-2:]
				}
				for _, branchCommit := range branchCommits {
					util.Fprintln(stdIo.Out, padding+"   - "+branchCommit.Subject)
				}
			}
		}
	}
}

func getLogsAndBranches() ([]templates.GitLog, []string) {
	logs := templates.GetNewCommits("HEAD")
	gitBranchArgs := make([]string, 0, len(logs)+2)
	gitBranchArgs = append(gitBranchArgs, "branch", "-l")
	for _, log := range logs {
		gitBranchArgs = append(gitBranchArgs, log.Branch)
	}
	checkedBranches := strings.Fields(util.ExecuteOrDie(util.ExecuteOptions{}, "git", gitBranchArgs...))
	return logs, checkedBranches
}
