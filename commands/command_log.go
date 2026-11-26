package commands

import (
	"fmt"
	"log/slog"
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
			"A " + color.GreenString("✓") + " means that there is a PR associated with the commit (actually it\n" +
			"means there is a branch, but having a branch means there is a PR when\n" +
			"using this workflow). If there is more than one commit on the\n" +
			"associated branch, those commits are also listed (indented under\n" +
			"their associated commit summary).\n" +
			"\n" +
			"A " + color.RedString("✓") + " means the branch has not been pushed to the remote.\n" +
			"\n" +
			"A " + interactive.StackEmoji + " beside a commit means its branch is part of a `gh stack` stack.\n" +
			"The letter after it (A, B, ...) identifies the stack, starting from the\n" +
			"bottom of the list. Commits with the same letter are in the same stack.\n" +
			"The letter is omitted when there is only one stack.\n" +
			"For a branch above another branch in the stack, only the commits since the\n" +
			"merge-base with the branch below are listed, followed by a ⬇️ and the number of\n" +
			"the commit for the branch below (omitted when it is the row directly below; its\n" +
			"branch name if not in the list; ❓ if the merge-base could not be found).\n" +
			"\n" +
			"A " + color.YellowString("●") + " means that multiple commits have the same subject.\n" +
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
	main := cmd.Flags().BoolP("main", "m", false,
		"Show the log as if run from the local main branch, instead of the current branch.")
	cmd.Run = func(cmd *cobra.Command, args []string) {
		ref := "HEAD"
		if *main {
			ref = gitutil.GetLocalMainBranchOrDie()
		}
		if *status || *poll {
			printGitLogWithStatus(cmd, *poll, *main, ref)
		} else {
			printGitLog(*main, ref)
		}
	}
	return cmd
}

func printGitLogWithStatus(cmd *cobra.Command, poll bool, main bool, ref string) {
	if !main && util.GetCurrentBranchName() != gitutil.GetLocalMainBranchOrDie() {
		panic("--status is only supported on the main branch (or with --main)")
	}
	userConfig := util.GetUserConfig()
	var worktreeSections []interactive.WorktreeLogSection
	logs, checkedBranches := getLogsAndBranches(ref)
	if userConfig.ShowWorktrees {
		worktreeSections = getWorktreeSections(logs)
	}
	var pollInterval time.Duration
	if poll {
		pollInterval = userConfig.PollInterval
	}
	interactive.ShowLogStatus(logs, checkedBranches, pollInterval, func() ([]templates.GitLog, []string, []interactive.WorktreeLogSection) {
		return getLogsAndBranchesWithWorktrees(ref)
	}, worktreeSections)
}

// Prints changes in ref (the current branch, or local main if main is true) compared to the remote main branch to out.
func printGitLog(main bool, ref string) {
	stdIo := util.GetAppConfig().Io
	if !main && util.GetCurrentBranchName() != gitutil.GetLocalMainBranchOrDie() {
		gitArgs := []string{"--no-pager", "log", "--pretty=oneline", "--abbrev-commit", "--decorate"}
		mergeBase, previousBranch := getStackMergeBase()
		if mergeBase != "" {
			gitArgs = append(gitArgs, mergeBase+"..HEAD")
		} else if gitutil.RemoteHasBranch(gitutil.GetRemoteMainBranchOrDie()) {
			gitArgs = append(gitArgs, "origin/"+gitutil.GetRemoteMainBranchOrDie()+"..HEAD")
		}
		gitArgs = append(gitArgs, "--color=always")
		util.ExecuteOrDie(util.ExecuteOptions{Io: stdIo}, "git", gitArgs)
		if previousBranch != "" {
			util.Fprintln(stdIo.Out, templates.StackArrow+" "+previousBranch)
		}
		return
	}
	logs, checkedBranches := getLogsAndBranches(ref)
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

// getStackMergeBase returns the merge-base of HEAD and the previous branch in its `gh stack`, along with the
// name of that previous branch. Both are empty if the current branch is not above another branch in a stack.
// Only the merge-base is empty if it cannot be found.
func getStackMergeBase() (string, string) {
	branch := util.GetCurrentBranchName()
	stackBranches := gitutil.GetGhStackBranches(branch)
	index := slices.Index(stackBranches, branch)
	if index <= 0 {
		return "", ""
	}
	previous := stackBranches[index-1]
	mergeBase, err := util.Execute(util.ExecuteOptions{}, "git", "merge-base", previous, "HEAD")
	if err != nil {
		slog.Debug(fmt.Sprint("Could not find merge-base of ", previous, " and HEAD: ", err))
		return "", previous
	}
	return strings.TrimSpace(mergeBase), previous
}

// printLogs prints commit logs and returns true if any duplicate subjects were found.
func printLogs(stdIo util.StdIo, logs []templates.GitLog, checkedBranches []string) bool {
	hasDuplicates := false
	stackBranches := make([]string, len(logs))
	for i, log := range logs {
		if slices.Contains(checkedBranches, log.Branch) {
			stackBranches[i] = log.Branch
		}
	}
	stackLabels := gitutil.GetGhStackLabels(stackBranches)
	maxLabelLen := 0
	anyStacked := false
	for _, label := range stackLabels {
		anyStacked = anyStacked || label.InStack
		maxLabelLen = max(maxLabelLen, len(label.Label))
	}
	remoteBranches := gitutil.RemoteBranchSet()
	for i, log := range logs {
		numberPrefix := interactive.GetLogNumberPrefix(i, len(logs))
		hasPR := slices.Contains(checkedBranches, log.Branch)
		// The stack emoji is 2 columns wide, followed by its label, so other rows get extra spaces when any row has one.
		extraPad := ""
		if anyStacked {
			extraPad = strings.Repeat(" ", 1+maxLabelLen)
		}
		if log.HasDuplicate {
			hasDuplicates = true
			util.Fprint(stdIo.Out, numberPrefix+color.YellowString("● ")+extraPad)
		} else if stackLabels[i].InStack {
			util.Fprint(stdIo.Out, numberPrefix+interactive.StackEmoji+stackLabels[i].Label+strings.Repeat(" ", maxLabelLen-len(stackLabels[i].Label))+" ")
		} else if hasPR {
			util.Fprint(stdIo.Out, numberPrefix+interactive.PrCheckMark(remoteBranches[log.Branch])+" "+extraPad)
		} else {
			util.Fprint(stdIo.Out, numberPrefix+"  "+extraPad)
		}
		util.Fprintln(stdIo.Out, color.YellowString(log.Commit)+" "+log.Subject)
		if hasPR {
			padding := strings.Repeat(" ", len(numberPrefix)+len(extraPad))
			branchCommits, below := templates.GetBranchCommits(logs, i, stackLabels[i].InStack, "")
			if len(branchCommits) > 1 {
				util.Fprint(stdIo.Out, interactive.FormatBranchCommits(branchCommits, padding))
			}
			if below != "" {
				util.Fprintln(stdIo.Out, padding+"  "+below)
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

func getLogsAndBranches(ref string) ([]templates.GitLog, []string) {
	logs := templates.GetNewCommits(ref, "")
	branchNames := make([]string, len(logs))
	for i, log := range logs {
		branchNames[i] = log.Branch
	}
	checkedBranches := gitutil.CheckLocalBranches("", branchNames)
	return logs, checkedBranches
}

func getLogsAndBranchesWithWorktrees(ref string) ([]templates.GitLog, []string, []interactive.WorktreeLogSection) {
	logs, checkedBranches := getLogsAndBranches(ref)
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
