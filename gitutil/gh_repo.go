package gitutil

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// Returns "repository-owner/repository-name" for the origin remote.
// Uses the origin URL explicitly so that forks resolve to the fork itself,
// not the parent repository.
func GetRepoNameWithOwner() string {
	cache.repoNameWithOwnerOnce.Do(func() {
		originURL := getOriginURL()
		cache.repoNameWithOwner = util.ExecuteOrDieTrimmed(util.ExecuteOptions{Retries: GhRetries},
			"gh", "repo", "view", originURL, "--json", "nameWithOwner", "--jq", ".nameWithOwner")
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

// GhRepoArgs returns "--repo", "owner/repo" for use with gh pr subcommands
// when in a fork, or an empty slice when not in a fork.
func GhRepoArgs() []string {
	if !isFork() {
		return []string{}
	}
	return []string{"--repo", GetRepoNameWithOwner()}
}

func isFork() bool {
	cache.isForkOnce.Do(func() {
		out, err := util.Execute(util.ExecuteOptions{}, "git", "remote", "get-url", "upstream")
		cache.isFork = err == nil
		slog.Debug(fmt.Sprint("isFork ", cache.isFork, ", upstream ", out))
	})
	return cache.isFork
}

func getOriginURL() string {
	return util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "remote", "get-url", "origin")
}
