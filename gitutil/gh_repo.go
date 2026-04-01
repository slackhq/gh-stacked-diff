package gitutil

import (
	"log/slog"
	"net/url"
	"strings"
	"sync"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// Cached repository name with owner.
var repoNameWithOwner string
var repoNameWithOwnerOnce *sync.Once = new(sync.Once)

// Cached logged in username
var loggedInUsername string
var loggedInUsernameOnce *sync.Once = new(sync.Once)

var cachedRepoHostname string = ""
var cachedRepoHostnameOnce *sync.Once = new(sync.Once)

// Returns "repository-owner/repository-name".
func GetRepoNameWithOwner() string {
	if repoNameWithOwner == "" {
		repoNameWithOwnerOnce.Do(func() {
			repoNameWithOwner = util.ExecuteOrDieTrimmed(util.ExecuteOptions{Retries: GhRetries},
				"gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
		})
	}
	return repoNameWithOwner
}

func GetLoggedInUsername() string {
	if loggedInUsername == "" {
		loggedInUsernameOnce.Do(func() {
			jq := ".hosts | to_entries[] | select(.key == \"" + GetRepoHostname() + "\") | .value[].login"
			out := util.ExecuteOrDie(util.ExecuteOptions{Retries: GhRetries},
				"gh", "auth", "status", "--active", "--json", "hosts", "--jq", jq)
			slog.Debug("loggedInUsername " + loggedInUsername)
			loggedInUsername = strings.Fields(out)[0]
		})
	}
	return loggedInUsername
}

func GetRepoHostname() string {
	if cachedRepoHostname == "" {
		cachedRepoHostnameOnce.Do(func() {
			out := util.ExecuteOrDieTrimmed(util.ExecuteOptions{Retries: GhRetries},
				"gh", "repo", "view", "--json", "url", "--jq", ".url")
			parsedUrl, err := url.Parse(out)
			if err != nil {
				panic("Could not parse url (" + out + "): " + err.Error())
			}
			cachedRepoHostname = parsedUrl.Hostname()
		})
	}
	return cachedRepoHostname
}
