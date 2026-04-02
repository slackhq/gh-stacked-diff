package interactive

import (
	"strings"

	"slices"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

/*
Example output of gh api collaborators:
gh api repos/{owner}/{repo}/collaborators
[

	{
	  "login": "xxx",
	  "id": 1234567,
	  "node_id": "XXX=",
	  "avatar_url": "https://avatars.githubusercontent.com/u/1234567?v=4",
	  "gravatar_id": "",
	  "url": "https://api.github.com/users/user-one",
	  "html_url": "https://github.com/user-one",
	  "followers_url": "https://api.github.com/users/user-one/followers",
	  "following_url": "https://api.github.com/users/user-one/following{/other_user}",
	  "gists_url": "https://api.github.com/users/user-one/gists{/gist_id}",
	  "starred_url": "https://api.github.com/users/user-one/starred{/owner}{/repo}",
	  "subscriptions_url": "https://api.github.com/users/user-one/subscriptions",
	  "organizations_url": "https://api.github.com/users/user-one/orgs",
	  "repos_url": "https://api.github.com/users/user-one/repos",
	  "events_url": "https://api.github.com/users/user-one/events{/privacy}",
	  "received_events_url": "https://api.github.com/users/user-one/received_events",
	  "type": "User",
	  "user_view_type": "public",
	  "site_admin": false,
	  "permissions": {
	    "admin": true,
	    "maintain": true,
	    "push": true,
	    "triage": true,
	    "pull": true
	  },
	  "role_name": "admin"
	},
	{
	  "login": "xxxx",
	  "id": 7654321,
	  "node_id": "YYY=",
	  "avatar_url": "https://avatars.githubusercontent.com/u/7654321?v=4",
	  "gravatar_id": "",
	  "url": "https://api.github.com/users/user-two",
	  "html_url": "https://github.com/user-two",
	  "followers_url": "https://api.github.com/users/user-two/followers",
	  "following_url": "https://api.github.com/users/user-two/following{/other_user}",
	  "gists_url": "https://api.github.com/users/user-two/gists{/gist_id}",
	  "starred_url": "https://api.github.com/users/user-two/starred{/owner}{/repo}",
	  "subscriptions_url": "https://api.github.com/users/user-two/subscriptions",
	  "organizations_url": "https://api.github.com/users/user-two/orgs",
	  "repos_url": "https://api.github.com/users/user-two/repos",
	  "events_url": "https://api.github.com/users/user-two/events{/privacy}",
	  "received_events_url": "https://api.github.com/users/user-two/received_events",
	  "type": "User",
	  "user_view_type": "public",
	  "site_admin": false,
	  "permissions": {
	    "admin": false,
	    "maintain": false,
	    "push": true,
	    "triage": true,
	    "pull": true
	  },
	  "role_name": "write"
	}

]

Example output from: gh repo view --json nameWithOwner

	{
	  "nameWithOwner": "{owner}/{repo}"
	}
*/
func getAllCollaborators() []string {
	jq := ".[] | .login"
	out := util.ExecuteOrDie(util.ExecuteOptions{Retries: gitutil.GhRetries},
		"gh", "api", "--hostname", gitutil.GetRepoHostname(), "repos/"+gitutil.GetRepoNameWithOwner()+"/collaborators",
		"--paginate", "--cache", "6h", "--jq", jq)
	collaborators := strings.Fields(out)
	collaborators = removeCurrentUser(collaborators)
	slices.Sort(collaborators)
	return collaborators
}

func removeCurrentUser(users []string) []string {
	return slices.DeleteFunc(users, func(next string) bool {
		return next == gitutil.GetLoggedInUsername()
	})
}
