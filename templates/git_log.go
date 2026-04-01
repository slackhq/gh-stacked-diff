package templates

import (
	"slices"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// Returned by some of the Get*Commit functions.
type GitLog struct {
	// Abbreviated commit hash.
	Commit string
	// Commit subject.
	Subject string
	// Associated branch name. Branch might not exist.
	Branch string
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
	logsRaw := util.ExecuteOrDie(util.ExecuteOptions{}, "git", gitArgs...)
	return newGitLogs(logsRaw)
}

func GetNewCommits(to string) []GitLog {
	compareFromRemoteBranch := gitutil.GetRemoteMainBranchOrDie()
	gitArgs := []string{"--no-pager", "log", newGitLogsFormat, "--abbrev-commit"}
	if gitutil.RemoteHasBranch(compareFromRemoteBranch) {
		gitArgs = append(gitArgs, "origin/"+compareFromRemoteBranch+".."+to)
	} else {
		gitArgs = append(gitArgs, to)
	}
	logsRaw := util.ExecuteOrDie(util.ExecuteOptions{}, "git", gitArgs...)
	return newGitLogs(logsRaw)
}

func newGitLogs(logsRaw string) []GitLog {
	logLines := strings.Split(strings.TrimSpace(logsRaw), "\n")
	var logs []GitLog
	for _, logLine := range logLines {
		components := strings.Split(logLine, formatDelimiter)
		if len(components) != 3 {
			// No git logs.
			continue
		}
		logs = append(logs, GitLog{Commit: components[0], Subject: components[1], Branch: getBranchForSantizedSubject(components[2])})
	}
	return logs
}

func RequireCommitOnMain(commit string) {
	if commit == gitutil.GetLocalMainBranchOrDie() {
		return
	}
	newCommits := GetNewCommits("HEAD")
	if !slices.ContainsFunc(newCommits, func(gitLog GitLog) bool {
		return gitLog.Commit == commit
	}) {
		panic("Commit " + commit + " does not exist on " + gitutil.GetLocalMainBranchOrDie() + ". Check `sd log` for available commits.")
	}
}
