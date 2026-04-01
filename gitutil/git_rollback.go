package gitutil

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

type restoreBranchInfo struct {
	commit string
	branch string
}
type GitRollbackManager struct {
	restoreBranches []restoreBranchInfo
	deleteBranches  []string
	skipRestore     bool
}

func (rollbackManager *GitRollbackManager) SaveState() {
	restoreBranch := restoreBranchInfo{
		commit: util.ExecuteOrDie(util.ExecuteOptions{}, "git", "log", "-n", "1", "--pretty=format:%H"),
		branch: util.GetCurrentBranchName(),
	}
	rollbackManager.restoreBranches = append(rollbackManager.restoreBranches, restoreBranch)
}

func (rollbackManager *GitRollbackManager) SkipRestore() {
	rollbackManager.skipRestore = true
}

func (rollbackManager *GitRollbackManager) Restore(err any) {
	if rollbackManager.skipRestore {
		return
	}
	if len(rollbackManager.restoreBranches) == 0 {
		// Nothing to restore.
		return
	}
	firstErrorLine := strings.Split(fmt.Sprint(err), "\n")[0]
	slog.Error("Restoring to original state because of error: " + firstErrorLine)
	tryAbort("cherry-pick")
	tryAbort("rebase")
	tryAbort("merge")
	for _, branchInfo := range slices.Backward(rollbackManager.restoreBranches) {
		slog.Info(fmt.Sprint("Restoring branch ", branchInfo.branch, " to ", branchInfo.commit))
		GitSwitch(branchInfo.branch)
		util.ExecuteOrDie(util.ExecuteOptions{}, "git", "reset", "--hard", branchInfo.commit)
	}
	for _, branch := range rollbackManager.deleteBranches {
		slog.Info(fmt.Sprint("Deleting created branch ", branch))
		util.ExecuteOrDie(util.ExecuteOptions{}, "git", "branch", "-D", branch)
	}
}

// Abort the given git command if it is in progress.
func tryAbort(gitCommand string) {
	_, err := util.Execute(util.ExecuteOptions{}, "git", gitCommand, "--abort")
	if err == nil {
		slog.Info(fmt.Sprint("Aborted ", gitCommand))
	}
}

func (rollbackManager *GitRollbackManager) CreatedBranch(branchName string) {
	rollbackManager.deleteBranches = append(rollbackManager.deleteBranches, branchName)
}

func (rollbackManager *GitRollbackManager) Clear() {
	rollbackManager.restoreBranches = []restoreBranchInfo{}
	rollbackManager.deleteBranches = []string{}
}

// WithStashAndRollback wraps the common pattern of stashing, saving rollback state,
// executing a function, and restoring on error. The callback receives the rollback
// manager so it can call SaveState() or CreatedBranch() as needed.
func WithStashAndRollback(stashName string, fn func(rollbackManager *GitRollbackManager)) {
	shouldPopStash := Stash(stashName)
	rollbackManager := &GitRollbackManager{}
	rollbackManager.SaveState()
	defer func() {
		r := recover()
		if r != nil {
			rollbackManager.Restore(r)
		}
		if !rollbackManager.skipRestore {
			PopStash(shouldPopStash)
		}
		if r != nil {
			panic(r)
		}
	}()
	fn(rollbackManager)
}
