package gitutil

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// gitCache holds all cached values that are expensive to recompute.
// Replaced atomically in tests via ResetCacheForTesting.
type gitCache struct {
	remoteMainBranch      string
	remoteMainOnce        sync.Once
	localMainBranch       string
	localMainOnce         sync.Once
	worktrees             []WorktreeInfo
	worktreesOnce         sync.Once
	mainBranchForHelp     string
	mainBranchForHelpOnce sync.Once
	emailLocalPart        string
	emailLocalPartOnce    sync.Once
	minChecks             int
	minChecksOnce         sync.Once
	// gh_repo.go caches
	repoNameWithOwner     string
	repoNameWithOwnerOnce sync.Once
	loggedInUsername      string
	loggedInUsernameOnce  sync.Once
	repoHostname          string
	repoHostnameOnce      sync.Once
}

var cache = &gitCache{}

// Returns name of the remote main branch (e.g. "main" or "master"), as determined
// from origin/HEAD. Returns an error if it cannot be determined.
// The result is cached after the first successful call.
func getRemoteMainBranch() (string, error) {
	if cache.remoteMainBranch != "" {
		return cache.remoteMainBranch, nil
	}
	branch, err := util.Execute(util.ExecuteOptions{}, "git", "rev-parse", "--abbrev-ref", "origin/HEAD")
	if err != nil {
		return "", err
	}
	branch = strings.TrimSpace(branch)
	_, after, found := strings.Cut(branch, "/")
	if !found {
		return "", fmt.Errorf("unexpected origin/HEAD format (no '/' in %q)", branch)
	}
	cache.remoteMainBranch = after
	return cache.remoteMainBranch, nil
}

// Returns name of the remote main branch, or panics if it cannot be determined.
// Handles initial repository setup (setting origin/HEAD) if needed.
// The result is cached after the first call.
func GetRemoteMainBranchOrDie() string {
	cache.remoteMainOnce.Do(func() {
		if _, err := getRemoteMainBranch(); err == nil {
			return
		}
		out, err := util.Execute(util.ExecuteOptions{}, "git", "rev-parse")
		if err != nil {
			panic("Not in a git repository. Must be run from a git repository.\n" + out + ": " + err.Error())
		}

		out, err = util.Execute(util.ExecuteOptions{}, "git", "rev-list", "--max-parents=0", "HEAD")
		if err != nil {
			panic("Remote repository is empty.\n" +
				"Push an initial inconsequential commit to origin/main and try again. \n" +
				"Using a repository without an initial remote commit is not recommended because git \n" +
				"requires special handling for the root commit, and you might accidentally \n" +
				"create more than one root commit.\n" + out + ": " + err.Error())
		}

		setRemoteHead()
		if _, err = getRemoteMainBranch(); err != nil {
			panic("Remote repository not setup: " + err.Error())
		}
	})
	return cache.remoteMainBranch
}

// Returns name of the local main branch. In a standard checkout or the main worktree,
// this is the same as the remote main branch. In a secondary worktree, this is the
// worktree's branch (the branch it was created with).
// The result is cached after the first successful call.
func getLocalMainBranch() (string, error) {
	if cache.localMainBranch != "" {
		return cache.localMainBranch, nil
	}
	if branch := getSecondaryWorktreeBranch(); branch != "" {
		cache.localMainBranch = branch
		return cache.localMainBranch, nil
	}
	branch, err := getRemoteMainBranch()
	if err != nil {
		return "", err
	}
	cache.localMainBranch = branch
	return cache.localMainBranch, nil
}

// Returns the current branch name if running in a secondary worktree,
// or empty string if in the main worktree or not using worktrees.
func getSecondaryWorktreeBranch() string {
	worktrees := GetWorktrees()
	if len(worktrees) <= 1 {
		return ""
	}
	currentRoot := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "rev-parse", "--show-toplevel")
	if currentRoot == worktrees[0].Path {
		return ""
	}
	if util.GetUserConfig().WorktreeMainBranchGuard == util.WorktreeMainBranchGuardNone {
		return util.GetCurrentBranchName()
	}
	return filepath.Base(currentRoot)
}

// Returns the path of the main worktree. Panics if it cannot be determined.
func GetMainWorktreePath() string {
	worktrees := GetWorktrees()
	if len(worktrees) == 0 {
		panic("Could not determine main worktree path")
	}
	return worktrees[0].Path
}

// Returns true if running in a secondary worktree (not the main worktree).
func IsSecondaryWorktree() bool {
	return getSecondaryWorktreeBranch() != ""
}

// WorktreeInfo holds the path and branch name (or commit hash for detached HEAD) of a git worktree.
type WorktreeInfo struct {
	Path           string
	BranchOrCommit string
}

// GetWorktrees returns all worktrees. The first entry (index 0) is always the
// main worktree. Panics if git worktree list fails.
// For worktrees on a detached HEAD, BranchOrCommit is set to the commit hash.
func GetWorktrees() []WorktreeInfo {
	cache.worktreesOnce.Do(func() {
		worktreeList := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "worktree", "list")
		lines := strings.Split(worktreeList, "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}
			path := fields[0]
			branchOrCommit := fields[1] // commit hash
			// Branch is in brackets, e.g. "[secondary-branch]"
			branchField := fields[2]
			if strings.HasPrefix(branchField, "[") {
				branchOrCommit = strings.Trim(branchField, "[]")
			}
			cache.worktrees = append(cache.worktrees, WorktreeInfo{Path: path, BranchOrCommit: branchOrCommit})
		}
	})
	return cache.worktrees
}

// GetSecondaryWorktrees returns all secondary worktrees (excludes the main worktree).
func GetSecondaryWorktrees() []WorktreeInfo {
	worktrees := GetWorktrees()
	if len(worktrees) <= 1 {
		return nil
	}
	return worktrees[1:]
}

// Returns name of the local main branch, or panics if it cannot be determined.
// The result is cached after the first call.
func GetLocalMainBranchOrDie() string {
	cache.localMainOnce.Do(func() {
		if _, err := getLocalMainBranch(); err == nil {
			return
		}
		// Fall through to GetRemoteMainBranchOrDie which handles setup.
		GetRemoteMainBranchOrDie()
		// Retry now that remote is set up.
		if _, err := getLocalMainBranch(); err != nil {
			panic("Could not determine local main branch: " + err.Error())
		}
	})
	return cache.localMainBranch
}

// Returns name of main branch, or "main" if cannot be determined. For use by CLI help.
func GetMainBranchForHelp() string {
	cache.mainBranchForHelpOnce.Do(func() {
		branch, err := getRemoteMainBranch()
		if err != nil {
			cache.mainBranchForHelp = "main"
		} else {
			cache.mainBranchForHelp = branch
		}
	})
	return cache.mainBranchForHelp
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

// GetEmailLocalPart returns the local-part (before the @) of the git user.email config value.
func GetEmailLocalPart() string {
	cache.emailLocalPartOnce.Do(func() {
		raw := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "config", "user.email")
		name, _, found := strings.Cut(raw, "@")
		if !found {
			panic("git config user.email does not contain '@': " + raw)
		}
		cache.emailLocalPart = name
	})
	return cache.emailLocalPart
}

// ApplyDiffFromRef diffs fromRef..toRef, applies the result to the working tree, and stages all changes.
func ApplyDiffFromRef(fromRef string, toRef string) {
	diff := util.ExecuteOrDie(util.ExecuteOptions{}, "git", "diff", "--binary", fromRef, toRef)
	util.ExecuteOrDie(
		util.ExecuteOptions{Io: util.StdIo{In: strings.NewReader(diff), Out: nil, Err: nil}},
		"git", "apply",
	)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")
}

// GetMergeBaseWithOriginMain returns the merge-base (divergence point) between branchName and origin/main.
func GetMergeBaseWithOriginMain(branchName string) string {
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

// CheckLocalBranches returns the subset of branchNames that exist as local branches.
func CheckLocalBranches(gitDir string, branchNames []string) []string {
	args := make([]string, 0, len(branchNames)+2)
	args = append(args, "branch", "-l")
	args = append(args, branchNames...)
	output := util.ExecuteOrDie(util.ExecuteOptions{}, "git", PrependGitDir(gitDir, args...)...)
	return strings.Fields(output)
}

// PrependGitDir prepends "-C", dir to args if dir is non-empty.
// This causes git to run as if invoked from that directory.
func PrependGitDir(dir string, args ...string) []string {
	if dir == "" {
		return args
	}
	return append([]string{"-C", dir}, args...)
}

// CherryPickAndSkipAllEmpty cherry-picks all commits and skips any that are empty
// (i.e., commits that are already on main and would result in no changes).
func CherryPickAndSkipAllEmpty(gitDir string, commits []string) {
	cherryPickArgs := make([]string, 2+len(commits))
	cherryPickArgs[0] = "cherry-pick"
	cherryPickArgs[1] = "--ff"
	for i, commit := range commits {
		cherryPickArgs[i+2] = commit
	}
	out, err := util.Execute(util.ExecuteOptions{}, "git", PrependGitDir(gitDir, cherryPickArgs...)...)
	for err != nil {
		if strings.Contains(out, "git commit --allow-empty") {
			slog.Debug("Skipping empty commit (already on main)")
			out, err = util.Execute(util.ExecuteOptions{}, "git", PrependGitDir(gitDir, "cherry-pick", "--skip")...)
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
			slog.Warn(fmt.Sprint("Failed to abort rebase: ", abortErr.Error()))
		}
		panic("Rebase failed: " + outRecorder.String() + " (" + err.Error() + ")")
	}
	return out
}

// Arguments to git that specify rebase with no-verify that works.
// --no-verify on its own does not propagate to commits that are done,
// so set the hooksPath to /dev/null.
// Bypassing hooks during rebase is safe because the resulting commits must
// be cherry-picked onto the PR branch via `sd new“ or `sd update` before
// they can be pushed, which would go through the git hooks.
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
