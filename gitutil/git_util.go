package gitutil

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// Cached value of main branch name for help text.
var mainBranchNameForHelp string

// Cached value of user email.
var userEmail string

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
	branch, err := util.Execute(util.ExecuteOptions{}, "git", "rev-parse", "--abbrev-ref", "origin/HEAD")
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
			out, err = util.Execute(util.ExecuteOptions{}, "git", "rev-parse")
			if err != nil {
				panic("Not in a git repository. Must be run from a git repository.\n" + out + ": " + err.Error())
			}

			out, err = util.Execute(util.ExecuteOptions{}, "git", "rev-list", "--max-parents=0", "HEAD")
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

// Returns name of the local main branch. In a standard checkout or the main worktree,
// this is the same as the remote main branch. In a secondary worktree, this is the
// worktree's branch (the branch it was created with).
// The result is cached after the first successful call.
func GetLocalMainBranch() (string, error) {
	if localMainBranch != "" {
		return localMainBranch, nil
	}
	if branch := getSecondaryWorktreeBranch(); branch != "" {
		localMainBranch = branch
		return localMainBranch, nil
	}
	branch, err := GetRemoteMainBranch()
	if err != nil {
		return "", err
	}
	localMainBranch = branch
	return localMainBranch, nil
}

// Returns the current branch name if running in a secondary worktree,
// or empty string if in the main worktree or not using worktrees.
func getSecondaryWorktreeBranch() string {
	worktreeList, err := util.Execute(util.ExecuteOptions{}, "git", "worktree", "list")
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(worktreeList), "\n")
	if len(lines) <= 1 {
		return ""
	}
	// First worktree listed is the main worktree.
	mainWorktreePath := strings.Fields(lines[0])[0]
	currentRoot := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "--show-toplevel")
	if currentRoot == mainWorktreePath {
		return ""
	}
	if util.GetUserConfig().WorktreeMainBranchGuard == util.WorktreeMainBranchGuardNone {
		return util.GetCurrentBranchName()
	}
	return filepath.Base(currentRoot)
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
	currentBranch := util.GetCurrentBranchName()
	defaultBranch, err := util.Execute(util.ExecuteOptions{}, "git", "config", "init.defaultBranch")
	if err != nil {
		// git config init.defaultBranch will fail if default branch is not setup.
		defaultBranch = "master"
	} else {
		defaultBranch = strings.TrimSpace(defaultBranch)
	}
	if currentBranch == defaultBranch || currentBranch == "main" {
		slog.Warn("Setting remote head to " + currentBranch + " because it is not set.")
		out, err := util.Execute(util.ExecuteOptions{}, "git", "remote", "set-head", "origin", currentBranch)
		if err != nil {
			panic("Remote repository not setup.\n" + out)
		}
	} else {
		panic("Remote head is not set, and it cannot be set automatically because current branch is not default (" + defaultBranch + ") or main.")
	}
}

func GetUsername() string {
	if userEmail == "" {
		userEmailRaw := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "config", "user.email")
		userEmail = userEmailRaw[0:strings.Index(userEmailRaw, "@")]
	}
	return userEmail
}

// Returns most recent commit of the given branch that is on origin/main.
func FirstOriginMainCommit(branchName string) string {
	if !GetLocalHasBranchOrDie(branchName) {
		panic("Branch does not exist " + branchName)
	}
	return util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "merge-base", "origin/"+GetRemoteMainBranchOrDie(), branchName)
}

// Returns whether branchName is on remote.
func RemoteHasBranch(branchName string) bool {
	remoteBranch := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "branch", "-r", "--list", "origin/"+branchName)
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
	out, err := util.Execute(util.ExecuteOptions{}, "git", "branch", "--list", branchName)
	if err != nil {
		return false, err
	}
	localBranch := strings.TrimSpace(out)
	return localBranch != "", nil
}

func RequireMainBranch() {
	if util.GetCurrentBranchName() != GetLocalMainBranchOrDie() {
		panic("Must be run from " + GetLocalMainBranchOrDie() + " branch\nTo disable this check set config worktreeMainBranchGuard to none")
	}
}

func GitSwitch(branchName string) {
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", branchName)
}

func Stash(forName string) bool {
	stashResult := strings.Split(util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "stash", "save", "-u", "before "+forName), "\n")
	if len(stashResult) > 0 && strings.HasPrefix(stashResult[len(stashResult)-1], "Saved working") {
		slog.Info(stashResult[len(stashResult)-1])
		return true
	}
	return false
}

func PopStash(popStash bool) {
	if popStash {
		util.ExecuteOrDie(util.ExecuteOptions{}, "git", "stash", "pop")
		slog.Info("Popped stash back")
	}
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
	out, err := util.Execute(util.ExecuteOptions{}, "git", cherryPickArgs...)
	for err != nil {
		if strings.Contains(out, "git commit --allow-empty") {
			slog.Debug("Skipping empty commit (already on main)")
			out, err = util.Execute(util.ExecuteOptions{}, "git", "cherry-pick", "--skip")
		} else {
			slog.Error("Unexpected cherry-pick error: " + out)
			panic("Unexpected cherry-pick error: " + out + " args: " + strings.Join(cherryPickArgs, " ") + " error: " + err.Error())
		}
	}
}

func RebaseAndSkipAllEmpty(options util.ExecuteOptions, otherRebaseArgs ...string) (string, error) {
	outRecorder := util.NewWriteRecorder(options.Io.Out)
	options.Io.Out = outRecorder
	options.Io.Err = outRecorder
	out, err := util.Execute(options,
		"git", append(rebaseNoVerify(), otherRebaseArgs...)...)
	for err != nil {
		if strings.Contains(outRecorder.String(), "git commit --allow-empty") {
			slog.Debug("Skipping empty commit (already on main)")
			out, err = util.Execute(options, "git", "rebase", "--skip")
		} else {
			break
		}
	}
	return out, err
}

func RebaseAndSkipAllEmptyOrDie(options util.ExecuteOptions, otherRebaseArgs ...string) string {
	outRecorder := util.NewWriteRecorder(options.Io.Out)
	options.Io.Out = outRecorder
	options.Io.Err = outRecorder
	out, err := RebaseAndSkipAllEmpty(options, otherRebaseArgs...)
	if err != nil {
		if _, abortErr := util.Execute(util.ExecuteOptions{}, "git", "rebase", "--abort"); abortErr != nil {
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
