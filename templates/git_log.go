package templates

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// DuplicateSubjectLegend is the warning shown when multiple commits share the same subject.
const DuplicateSubjectLegend = "● indicates that multiple commits have the same subject. Change the subjects to differentiate commits"

// Returned by some of the Get*Commit functions.
type GitLog struct {
	// Abbreviated commit hash.
	Commit string
	// Commit subject.
	Subject string
	// Associated branch name. Branch might not exist.
	Branch string
	// Whether another commit in the same log has the same subject.
	HasDuplicate bool
}

func (g GitLog) String() string {
	return g.Commit + " " + g.Subject
}

// Delimiter for git log format when a space cannot be used.
const formatDelimiter = "|stackeddiff-delim|"

// Format sent to "git log" for use by [newGitLogs].
// %h = abbrieviated commit | %s = summary | %f = summary that can be used as filename
const newGitLogsFormat = "--pretty=format:%h" + formatDelimiter + "%s" + formatDelimiter + "%f"

// Returns all the commits on the current branch. For use by tests.
func GetAllCommits() []GitLog {
	gitArgs := []string{"--no-pager", "log", newGitLogsFormat, "--abbrev-commit"}
	logsRaw := util.ExecuteOrDie(util.ExecuteOptions{}, "git", gitArgs)
	return newGitLogs(logsRaw)
}

func GetNewCommits(to string, gitDir string) []GitLog {
	compareFromRemoteBranch := gitutil.GetRemoteMainBranchOrDie()
	gitArgs := []string{"--no-pager", "log", newGitLogsFormat, "--abbrev-commit"}
	if gitutil.RemoteHasBranch(compareFromRemoteBranch) {
		gitArgs = append(gitArgs, "origin/"+compareFromRemoteBranch+".."+to)
	} else {
		gitArgs = append(gitArgs, to)
	}
	logsRaw := util.ExecuteOrDie(util.ExecuteOptions{}, "git", gitutil.PrependGitDir(gitDir, gitArgs...))
	return newGitLogs(logsRaw)
}

// StackArrow ends the commit list of a branch that is above another branch in a `gh stack`.
const StackArrow = "⬇️"

// GetBranchCommits returns the commits to show beneath logs[i] (newest first). For a branch that is above another
// branch in a `gh stack`, these are the commits since the merge-base with the previous branch, and the second
// return value is the "⬇️ <target>" line to print last, where target is the list number of the commit for the
// previous branch (omitted if it is the row directly below, or its branch name if it is not in logs), or ❓ if the merge-base could not be determined.
// Otherwise the commits are those since the remote main branch and the second return value is empty.
func GetBranchCommits(logs []GitLog, i int, inStack bool, gitDir string) ([]GitLog, string) {
	branch := logs[i].Branch
	if inStack {
		stackBranches := gitutil.GetGhStackBranches(branch)
		if index := slices.Index(stackBranches, branch); index > 0 {
			previous := stackBranches[index-1]
			mergeBase, err := util.Execute(util.ExecuteOptions{}, "git", gitutil.PrependGitDir(gitDir, "merge-base", previous, branch))
			if err != nil {
				slog.Debug(fmt.Sprint("Could not find merge-base of ", previous, " and ", branch, ": ", err))
				return GetNewCommits(branch, gitDir), StackArrow + " ❓"
			}
			target := previous
			if j := slices.IndexFunc(logs, func(l GitLog) bool { return l.Branch == previous }); j >= 0 {
				target = fmt.Sprint(j + 1)
				if j == i+1 {
					// The commit directly below is the one being pointed to, so there is no need to name it.
					target = ""
				}
			}
			return GetCommitsSince(strings.TrimSpace(mergeBase), branch, gitDir), strings.TrimSpace(StackArrow + " " + target)
		}
	}
	return GetNewCommits(branch, gitDir), ""
}

// GetCommitsSince returns the commits reachable from "to" but not from "from", newest first.
func GetCommitsSince(from string, to string, gitDir string) []GitLog {
	logsRaw := util.ExecuteOrDie(util.ExecuteOptions{}, "git", gitutil.PrependGitDir(gitDir, "--no-pager", "log", newGitLogsFormat, "--abbrev-commit", from+".."+to))
	return newGitLogs(logsRaw)
}

func newGitLogs(logsRaw string) []GitLog {
	logLines := strings.Split(strings.TrimSpace(logsRaw), "\n")
	var logs []GitLog
	// Track the first index where each subject was seen to mark duplicates in a single pass.
	firstSeen := make(map[string]int, len(logLines))
	for _, logLine := range logLines {
		components := strings.Split(logLine, formatDelimiter)
		if len(components) != 3 {
			// No git logs.
			continue
		}
		log := GitLog{Commit: components[0], Subject: components[1], Branch: getBranchForSanitizedSubject(components[2])}
		if prevIndex, seen := firstSeen[log.Subject]; seen {
			log.HasDuplicate = true
			logs[prevIndex].HasDuplicate = true
		} else {
			firstSeen[log.Subject] = len(logs)
		}
		logs = append(logs, log)
	}
	return logs
}

func RequireCommitOnMain(commit string) {
	newCommits := GetNewCommits("HEAD", "")
	if !slices.ContainsFunc(newCommits, func(gitLog GitLog) bool {
		return gitLog.Commit == commit
	}) {
		panic(commitNotOnMainMessage(commit))
	}
}

func commitNotOnMainMessage(commit string) string {
	return "Commit " + commit + " does not exist on " + gitutil.GetLocalMainBranchOrDie() + ". Check `sd log` for available commits."
}

// SortEarliestFirst returns a copy of commits ordered earliest first, so that they can be cherry-picked in order.
//
// Panics if a commit is not one of the new commits (see [GetNewCommits]).
func SortEarliestFirst(commits []GitLog, gitDir string) []GitLog {
	// New commits are ordered newest first, so a higher index is an earlier commit.
	newCommits := GetNewCommits("HEAD", gitDir)
	indexOf := func(commit GitLog) int {
		index := slices.IndexFunc(newCommits, func(newCommit GitLog) bool {
			return newCommit.Commit == commit.Commit
		})
		if index == -1 {
			panic(commitNotOnMainMessage(commit.Commit))
		}
		return index
	}
	// Look up every commit first, as sorting does not compare a lone commit.
	indexes := make(map[string]int, len(commits))
	for _, commit := range commits {
		indexes[commit.Commit] = indexOf(commit)
	}
	sorted := slices.Clone(commits)
	slices.SortStableFunc(sorted, func(a, b GitLog) int {
		return indexes[b.Commit] - indexes[a.Commit]
	})
	return sorted
}
