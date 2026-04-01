package util

import (
	"path/filepath"
	"sync"
)

// Cached repository name.
var repoName string
var repoNameOnce *sync.Once = new(sync.Once)

// Returns current branch name.
func GetCurrentBranchName() string {
	return ExecuteOrDieTrimmed(ExecuteOptions{}, "git", "rev-parse", "--abbrev-ref", "HEAD")
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
