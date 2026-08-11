package util

import (
	"path/filepath"
	"strings"
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
	repoNameOnce.Do(func() {
		repoName = getRepoNameFromRemote()
		if repoName == "" {
			out := ExecuteOrDieTrimmed(ExecuteOptions{},
				"git", "rev-parse", "--show-toplevel")
			_, repoName = filepath.Split(out)
		}
	})
	return repoName
}

func getRepoNameFromRemote() string {
	out, err := Execute(ExecuteOptions{}, "git", "remote", "--verbose")
	if err != nil {
		return ""
	}
	// Use the origin remote's push URL. There may be multiple (push) lines
	// when working with forks (one per remote).
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if fields[0] == "origin" && strings.HasSuffix(line, "(push)") {
			return repoNameFromURL(fields[1])
		}
	}
	return ""
}

func repoNameFromURL(rawURL string) string {
	rawURL = strings.TrimSuffix(rawURL, ".git")
	// Split on both "/" and ":" to handle HTTPS (https://host/owner/repo)
	// and SSH (git@host:owner/repo) URLs uniformly.
	parts := strings.FieldsFunc(rawURL, func(r rune) bool {
		return r == '/' || r == ':'
	})
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-2] + "/" + parts[len(parts)-1]
}
