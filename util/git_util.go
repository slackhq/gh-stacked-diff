package util

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
)

// Cached value of main branch name for help text.
var mainBranchNameForHelp string

// Cached value of user email.
var userEmail string

// Cached repository name.
var repoName string
var repoNameOnce *sync.Once = new(sync.Once)

// Cached remote main branch name (from origin/HEAD).
var remoteMainBranch string
var remoteMainBranchOnce *sync.Once = new(sync.Once)

// Cached local main branch name.
var localMainBranch string
var localMainBranchOnce *sync.Once = new(sync.Once)

// Returns name of the remote main branch (e.g. "main" or "master"), as determined
// from origin/HEAD. Returns an error if it cannot be determined.
// The result is cached after the first successful call.
func GetRemoteMainBranch() (string, error) {
	if remoteMainBranch != "" {
		return remoteMainBranch, nil
	}
	branch, err := Execute(ExecuteOptions{}, "git", "rev-parse", "--abbrev-ref", "origin/HEAD")
	if err != nil {
		return "", err
	}
	branch = strings.TrimSpace(branch)
	remoteMainBranch = branch[strings.Index(branch, "/")+1:]
	return remoteMainBranch, nil
}

// Returns name of the remote main branch, or panics if it cannot be determined.
// Handles initial repository setup (setting origin/HEAD) if needed.
// The result is cached after the first call.
func GetRemoteMainBranchOrDie() string {
	if remoteMainBranch == "" {
		remoteMainBranchOnce.Do(func() {
			out, err := GetRemoteMainBranch()
			if err == nil {
				remoteMainBranch = out
				return
			}
			out, err = Execute(ExecuteOptions{}, "git", "rev-parse")
			if err != nil {
				panic("Not in a git repository. Must be run from a git repository.\n" + out + ": " + err.Error())
			}

			out, err = Execute(ExecuteOptions{}, "git", "rev-list", "--max-parents=0", "HEAD")
			if err != nil {
				panic("Remote repository is empty.\n" +
					"Push an initial inconsequential commit to origin/main and try again. \n" +
					"Using a repository without an initial remote commit is not recommended because git \n" +
					"requires special handling for the root commit, and you might accidentially \n" +
					"create more than one root commit.\n" + out + ": " + err.Error())
			}

			setRemoteHead()
			out, err = GetRemoteMainBranch()
			if err != nil {
				panic("Remote repository not setup.\n" + out + ": " + err.Error())
			}
			remoteMainBranch = out
		})
	}
	return remoteMainBranch
}

// Returns name of the local main branch. In a standard checkout this is the same
// as the remote main branch. In a secondary worktree, this will be the worktree's branch.
// The result is cached after the first successful call.
func GetLocalMainBranch() (string, error) {
	if localMainBranch != "" {
		return localMainBranch, nil
	}
	// For now, delegate to remote. Worktree support will be added here.
	branch, err := GetRemoteMainBranch()
	if err != nil {
		return "", err
	}
	localMainBranch = branch
	return localMainBranch, nil
}

// Returns name of the local main branch, or panics if it cannot be determined.
// The result is cached after the first call.
func GetLocalMainBranchOrDie() string {
	if localMainBranch == "" {
		localMainBranchOnce.Do(func() {
			out, err := GetLocalMainBranch()
			if err == nil {
				localMainBranch = out
				return
			}
			// Fall through to GetRemoteMainBranchOrDie which handles setup.
			GetRemoteMainBranchOrDie()
			// Retry now that remote is set up.
			result, err := GetLocalMainBranch()
			if err != nil {
				panic("Could not determine local main branch: " + err.Error())
			}
			localMainBranch = result
		})
	}
	return localMainBranch
}

// Returns name of main branch, or "main" if cannot be determined. For use by CLI help.
func GetMainBranchForHelp() string {
	if mainBranchNameForHelp != "" {
		return mainBranchNameForHelp
	}
	branch, err := GetRemoteMainBranch()
	if err != nil {
		mainBranchNameForHelp = "main"
	} else {
		mainBranchNameForHelp = branch
	}
	return mainBranchNameForHelp
}

func setRemoteHead() {
	currentBranch := GetCurrentBranchName()
	defaultBranch, err := Execute(ExecuteOptions{}, "git", "config", "init.defaultBranch")
	if err != nil {
		// git config init.defaultBranch will fail if default branch is not setup.
		defaultBranch = "master"
	} else {
		defaultBranch = strings.TrimSpace(defaultBranch)
	}
	if currentBranch == defaultBranch || currentBranch == "main" {
		slog.Warn("Setting remote head to " + currentBranch + " because it is not set.")
		out, err := Execute(ExecuteOptions{}, "git", "remote", "set-head", "origin", currentBranch)
		if err != nil {
			panic("Remote repository not setup.\n" + out)
		}
	} else {
		panic("Remote head is not set, and it cannot be set automatically because current branch is not default (" + defaultBranch + ") or main.")
	}
}

func GetUsername() string {
	if userEmail == "" {
		userEmailRaw := ExecuteOrDieTrimmed(ExecuteOptions{}, "git", "config", "user.email")
		userEmail = userEmailRaw[0:strings.Index(userEmailRaw, "@")]
	}
	return userEmail
}

// Returns most recent commit of the given branch that is on origin/main.
func FirstOriginMainCommit(branchName string) string {
	if !GetLocalHasBranchOrDie(branchName) {
		panic("Branch does not exist " + branchName)
	}
	return ExecuteOrDieTrimmed(ExecuteOptions{}, "git", "merge-base", "origin/"+GetRemoteMainBranchOrDie(), branchName)
}

// Returns whether branchName is on remote.
func RemoteHasBranch(branchName string) bool {
	remoteBranch := ExecuteOrDieTrimmed(ExecuteOptions{}, "git", "branch", "-r", "--list", "origin/"+branchName)
	return remoteBranch != ""
}

func GetLocalHasBranchOrDie(branchName string) bool {
	hasBranch, err := localHasBranch(branchName)
	if err != nil {
		panic(err)
	}
	return hasBranch
}

func localHasBranch(branchName string) (bool, error) {
	out, err := Execute(ExecuteOptions{}, "git", "branch", "--list", branchName)
	if err != nil {
		return false, err
	}
	localBranch := strings.TrimSpace(out)
	return localBranch != "", nil
}

func RequireMainBranch() {
	if GetCurrentBranchName() != GetLocalMainBranchOrDie() {
		panic("Must be run from " + GetLocalMainBranchOrDie() + " branch")
	}
}

// Returns current branch name.
func GetCurrentBranchName() string {
	return ExecuteOrDieTrimmed(ExecuteOptions{}, "git", "rev-parse", "--abbrev-ref", "HEAD")
}

func GitSwitch(branchName string) {
	ExecuteOrDie(ExecuteOptions{}, "git", "switch", branchName)
}

func Stash(forName string) bool {
	stashResult := strings.Split(ExecuteOrDieTrimmed(ExecuteOptions{}, "git", "stash", "save", "-u", "before "+forName), "\n")
	if len(stashResult) > 0 && strings.HasPrefix(stashResult[len(stashResult)-1], "Saved working") {
		slog.Info(stashResult[len(stashResult)-1])
		return true
	}
	return false
}

func PopStash(popStash bool) {
	if popStash {
		ExecuteOrDie(ExecuteOptions{}, "git", "stash", "pop")
		slog.Info("Popped stash back")
	}
}

func GetRepoName() string {
	if repoName == "" {
		repoNameOnce.Do(func() {
			out := ExecuteOrDieTrimmed(ExecuteOptions{},
				"git", "rev-parse", "--show-toplevel")
			_, repoName = filepath.Split(out)
		})
	}
	return repoName
}

// CherryPickAndSkipAllEmpty cherry-picks all commits and skips any that are empty
// (i.e., commits that are already on main and would result in no changes).
func CherryPickAndSkipAllEmpty(commits []string) {
	cherryPickArgs := make([]string, 2+len(commits))
	cherryPickArgs[0] = "cherry-pick"
	cherryPickArgs[1] = "--ff"
	for i, commit := range commits {
		cherryPickArgs[i+2] = commit
	}
	out, err := Execute(ExecuteOptions{}, "git", cherryPickArgs...)
	for err != nil {
		if strings.Contains(out, "git commit --allow-empty") {
			slog.Debug("Skipping empty commit (already on main)")
			out, err = Execute(ExecuteOptions{}, "git", "cherry-pick", "--skip")
		} else {
			slog.Error("Unexpected cherry-pick error: " + out)
			panic("Unexpected cherry-pick error: " + out + " args: " + strings.Join(cherryPickArgs, " ") + " error: " + err.Error())
		}
	}
}

func RebaseAndSkipAllEmpty(options ExecuteOptions, otherRebaseArgs ...string) (string, error) {
	outRecorder := NewWriteRecorder(options.Io.Out)
	options.Io.Out = outRecorder
	options.Io.Err = outRecorder
	out, err := Execute(options,
		"git", append(rebaseNoVerify(), otherRebaseArgs...)...)
	for err != nil {
		if strings.Contains(outRecorder.String(), "git commit --allow-empty") {
			slog.Debug("Skipping empty commit (already on main)")
			out, err = Execute(options, "git", "rebase", "--skip")
		} else {
			break
		}
	}
	return out, err
}

func RebaseAndSkipAllEmptyOrDie(options ExecuteOptions, otherRebaseArgs ...string) string {
	outRecorder := NewWriteRecorder(options.Io.Out)
	options.Io.Out = outRecorder
	options.Io.Err = outRecorder
	out, err := RebaseAndSkipAllEmpty(options, otherRebaseArgs...)
	if err != nil {
		if _, abortErr := Execute(ExecuteOptions{}, "git", "rebase", "--abort"); abortErr != nil {
			slog.Warn(fmt.Sprintf("Failed to abort rebase: %s", abortErr.Error()))
		}
		panic("Rebase failed: " + outRecorder.String() + " (" + err.Error() + ")")
	}
	return out
}

// Arguments to git that specify rebase with no-verify that works.
// --no-verify on its own does not propagate to commits that are done,
// so set the hooksPath to /dev/null.
// Modified from https://superuser.com/a/1827815/681646
func rebaseNoVerify() []string {
	// Also use commit.cleanup=strip to avoid having conflict comments added to commit message.
	return []string{
		"-c", "core.hooksPath=/dev/null",
		"-c", "commit.cleanup=strip",
		"-c", "advice.skippedCherryPicks=false",
		"-c", "advice.diverging=false",
		"rebase", "--no-verify"}
}
