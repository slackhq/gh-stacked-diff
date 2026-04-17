package gitutil

import (
	"log/slog"
	"net/url"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// Returns "repository-owner/repository-name".
func GetRepoNameWithOwner() string {
	cache.repoNameWithOwnerOnce.Do(func() {
		cache.repoNameWithOwner = util.ExecuteOrDieTrimmed(util.ExecuteOptions{Retries: GhRetries},
			"gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	})
	return cache.repoNameWithOwner
}

func GetLoggedInUsername() string {
	cache.loggedInUsernameOnce.Do(func() {
		hostname := GetRepoHostname()
		util.RequireHostname(hostname)
		jq := ".hosts | to_entries[] | select(.key == \"" + hostname + "\") | .value[].login"
		out := util.ExecuteOrDie(util.ExecuteOptions{Retries: GhRetries},
			"gh", "auth", "status", "--active", "--json", "hosts", "--jq", jq)
		fields := strings.Fields(out)
		if len(fields) == 0 {
			panic("gh auth status returned no login for host " + hostname)
		}
		cache.loggedInUsername = fields[0]
		slog.Debug("loggedInUsername " + cache.loggedInUsername)
	})
	return cache.loggedInUsername
}

func GetRepoHostname() string {
	cache.repoHostnameOnce.Do(func() {
		out := util.ExecuteOrDieTrimmed(util.ExecuteOptions{Retries: GhRetries},
			"gh", "repo", "view", "--json", "url", "--jq", ".url")
		parsedUrl, err := url.Parse(out)
		if err != nil {
			panic("Could not parse url (" + out + "): " + err.Error())
		}
		cache.repoHostname = parsedUrl.Hostname()
	})
	return cache.repoHostname
}
