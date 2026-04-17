package commands

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
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
			"associated branch, those commits are also listed (indented under\n" +
			"their associated commit summary).\n" +
			"\n" +
			"A " + color.YellowString("🟡") + " means that multiple commits have the same subject.\n" +
			"Change the subjects to differentiate commits.",
		Args: cobra.NoArgs,
		Annotations: map[string]string{
			"defaultLogLevel":   "error",
			checkRepoAnnotation: "true",
		},
	}
	status := cmd.Flags().BoolP("status", "s", false,
		"Show PR status including checks, approvals, and state.\n"+
			"Only supported on the main branch.")
	poll := cmd.Flags().BoolP("poll", "p", false,
		"Keep polling for status updates. Implies --status.\n"+
			"Press Esc or Ctrl+C to exit.")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		if *status || *poll {
			printGitLogWithStatus(cmd, *poll)
		} else {
			printGitLog()
		}
	}
	return cmd
}

func printGitLogWithStatus(cmd *cobra.Command, poll bool) {
	if util.GetCurrentBranchName() != gitutil.GetLocalMainBranchOrDie() {
		panic("--status is only supported on the main branch")
	}
	userConfig := util.GetUserConfig()
	var worktreeSections []interactive.WorktreeLogSection
	logs, checkedBranches := getLogsAndBranches()
	if userConfig.ShowWorktrees {
		worktreeSections = getWorktreeSections(logs)
	}
	var pollInterval time.Duration
	if poll {
		pollInterval = userConfig.PollInterval
	}
	interactive.ShowLogStatus(logs, checkedBranches, pollInterval, getLogsAndBranchesWithWorktrees, worktreeSections)
}

// Prints changes in the current branch compared to the main branch to out.
func printGitLog() {
	stdIo := util.GetAppConfig().Io
	if util.GetCurrentBranchName() != gitutil.GetLocalMainBranchOrDie() {
		gitArgs := []string{"--no-pager", "log", "--pretty=oneline", "--abbrev-commit"}
		if gitutil.RemoteHasBranch(gitutil.GetRemoteMainBranchOrDie()) {
			gitArgs = append(gitArgs, "origin/"+gitutil.GetRemoteMainBranchOrDie()+"..HEAD")
		}
		gitArgs = append(gitArgs, "--color=always")
		util.ExecuteOrDie(util.ExecuteOptions{Io: stdIo}, "git", gitArgs...)
		return
	}
	logs, checkedBranches := getLogsAndBranches()
	hasDuplicates := printLogs(stdIo, logs, checkedBranches)
	if util.GetUserConfig().ShowWorktrees {
		if printWorktreeLogs(stdIo, logs) {
			hasDuplicates = true
		}
	}
	if hasDuplicates && util.GetUserConfig().ShowDuplicateSubjectLegend {
		util.IncrementLegendShownCount(util.LegendDuplicateSubject)
		util.Fprintln(stdIo.Out, color.YellowString(templates.DuplicateSubjectLegend))
	}
}

// printLogs prints commit logs and returns true if any duplicate subjects were found.
func printLogs(stdIo util.StdIo, logs []templates.GitLog, checkedBranches []string) bool {
	hasDuplicates := false
	for i, log := range logs {
		numberPrefix := interactive.GetLogNumberPrefix(i, len(logs))
		hasPR := slices.Contains(checkedBranches, log.Branch)
		if log.HasDuplicate {
			hasDuplicates = true
			util.Fprint(stdIo.Out, numberPrefix+color.YellowString("🟡 "))
		} else if hasPR {
			// Use color for ✅ otherwise in Git Bash on Windows it will appear as black and white.
			util.Fprint(stdIo.Out, numberPrefix+color.GreenString("✅ "))
		} else {
			util.Fprint(stdIo.Out, numberPrefix+"   ")
		}
		util.Fprintln(stdIo.Out, color.YellowString(log.Commit)+" "+log.Subject)
		if hasPR {
			branchCommits := templates.GetNewCommits(log.Branch, "")
			if len(branchCommits) > 1 {
				padding := strings.Repeat(" ", len(numberPrefix))
				util.Fprint(stdIo.Out, interactive.FormatBranchCommits(branchCommits, padding))
			}
		}
	}
	return hasDuplicates
}

// Returns info about worktrees other than the current one.
func getOtherWorktrees() []gitutil.WorktreeInfo {
	worktrees := gitutil.GetWorktrees()
	if len(worktrees) <= 1 {
		return nil
	}
	currentRoot := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "--show-toplevel")
	return util.FilterSlice(worktrees, func(wt gitutil.WorktreeInfo) bool {
		return wt.Path != currentRoot
	})
}

// Prints logs from other worktrees, excluding commits already in the current directory.
// Returns true if any duplicate subjects were found.
func printWorktreeLogs(stdIo util.StdIo, currentLogs []templates.GitLog) bool {
	hasDuplicates := false
	for _, section := range getWorktreeSections(currentLogs) {
		util.Fprintln(stdIo.Out, "")
		util.Fprintln(stdIo.Out, color.New(color.Bold).Sprint(section.DirName))
		if printLogs(stdIo, section.Logs, section.CheckedBranches) {
			hasDuplicates = true
		}
	}
	return hasDuplicates
}

func getLogsAndBranches() ([]templates.GitLog, []string) {
	logs := templates.GetNewCommits("HEAD", "")
	branchNames := make([]string, len(logs))
	for i, log := range logs {
		branchNames[i] = log.Branch
	}
	checkedBranches := gitutil.CheckLocalBranches("", branchNames)
	return logs, checkedBranches
}

func getLogsAndBranchesWithWorktrees() ([]templates.GitLog, []string, []interactive.WorktreeLogSection) {
	logs, checkedBranches := getLogsAndBranches()
	worktreeSections := getWorktreeSections(logs)
	return logs, checkedBranches, worktreeSections
}

func getWorktreeSections(currentLogs []templates.GitLog) []interactive.WorktreeLogSection {
	otherWorktrees := getOtherWorktrees()
	if len(otherWorktrees) == 0 {
		return nil
	}
	currentSubjects := make(map[string]bool, len(currentLogs))
	for _, log := range currentLogs {
		currentSubjects[log.Subject] = true
	}
	var sections []interactive.WorktreeLogSection
	for _, wt := range otherWorktrees {
		wtLogs := templates.GetNewCommits(wt.BranchOrCommit, "")
		var uniqueLogs []templates.GitLog
		for _, log := range wtLogs {
			if !currentSubjects[log.Subject] {
				uniqueLogs = append(uniqueLogs, log)
			}
		}
		if len(uniqueLogs) == 0 {
			continue
		}
		branchNames := make([]string, len(uniqueLogs))
		for i, log := range uniqueLogs {
			branchNames[i] = log.Branch
		}
		checkedBranches := gitutil.CheckLocalBranches("", branchNames)
		dirName := filepath.Base(wt.Path)
		if wt.BranchOrCommit != dirName {
			dirName += " (" + wt.BranchOrCommit + ")"
		}
		sections = append(sections, interactive.WorktreeLogSection{
			DirName:         dirName,
			Logs:            uniqueLogs,
			CheckedBranches: checkedBranches,
		})
	}
	return sections
}
