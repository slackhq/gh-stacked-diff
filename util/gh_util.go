package util

import (
	"slices"
	"strings"
)

const GhRetries = 2

type PullRequestState int

const (
	PullRequestStateOpen PullRequestState = iota
	PullRequestStateMerged
	PullRequestStateClosed
)

type PullRequestStatus struct {
	Checks    PullRequestChecksStatus
	Approvers []string
	State     PullRequestState
}

/*
Returns users that have already approved latest commit.

Example output of gh pr view:

$ gh pr view mybranch --json "reviews"

	{
	  "reviews": [
	    {
	      "id": "PRR_kwDODeVIac6f37Qq",
	      "author": {
	        "login": "mybestie"
	      },
	      "authorAssociation": "MEMBER",
	      "body": "",
	      "submittedAt": "2025-03-13T14:47:31Z",
	      "includesCreatedEdit": false,
	      "reactionGroups": [],
	      "state": "COMMENTED",
	      "commit": {
	        "oid": "af01bdf8eb5649956096a608717f7de5eeb97e45"
	      }
	    },
	    {
	      "id": "PRR_kwDODeVIac6f5jeG",
	      "author": {
	        "login": "myfave"
	      },
	      "authorAssociation": "MEMBER",
	      "body": "",
	      "submittedAt": "2025-03-13T16:32:44Z",
	      "includesCreatedEdit": false,
	      "reactionGroups": [],
	      "state": "APPROVED",
	      "commit": {
	        "oid": "af01bdf8eb5649956096a608717f7de5eeb97e45"
	      }
	    }
	  ]
	}
*/
func GetAllApprovingUsers(branchName string) []string {
	// Note: technically it is possible to query for more than one PR at a time but requires knowing a commit hash so not as reliable.
	// gh pr list --search "429bb20,0ff019b" --state all
	lastCommit := GetBranchLatestCommit(branchName)
	jq := ".reviews[] | select(.state == \"APPROVED\" and .commit.oid == \"" + lastCommit + "\") | .author.login"
	out := ExecuteOrDie(ExecuteOptions{Retries: GhRetries},
		"gh", "pr", "view", branchName, "--json", "reviews", "--jq", jq)
	approvingUsers := strings.Fields(out)
	slices.Sort(approvingUsers)
	return slices.Compact(approvingUsers)
}

// Returns full commit hash of branch with name of branchName, or "" if no such branch.
func GetBranchLatestCommit(branchName string) string {
	out, err := Execute(ExecuteOptions{}, "git", "log", "-n", "1", "--pretty=format:%H", branchName)
	if err != nil {
		return ""
	} else {
		return strings.TrimSpace(out)
	}
}

/*
$ gh pr view 73 --json "reviews,statusCheckRollup" --jq "pick(.reviews[].author.login, .reviews[].state, .reviews[].commit.oid, .statusCheckRollup[].status, .statusCheckRollup[].conclusion, .statusCheckRollup[].state)"

	{
	  "reviews": [
	    {
	      "author": {
	        "login": "jallentest1"
	      },
	      "commit": {
	        "oid": "b7a6a8e29a906fbb009e5747167c5d11e80bc9b3"
	      },
	      "state": "CHANGES_REQUESTED"
	    },
	    {
	      "author": {
	        "login": "jallentest1"
	      },
	      "commit": {
	        "oid": "b7a6a8e29a906fbb009e5747167c5d11e80bc9b3"
	      },
	      "state": "APPROVED"
	    },
	    {
	      "author": {
	        "login": "jallentest1"
	      },
	      "commit": {
	        "oid": "b7a6a8e29a906fbb009e5747167c5d11e80bc9b3"
	      },
	      "state": "COMMENTED"
	    }
	  ],
	  "statusCheckRollup": [
	    {
	      "conclusion": "SUCCESS",
	      "state": null,
	      "status": "COMPLETED"
	    }
	  ]
	}
*/
func GetPullRequestStatus(appConfig AppConfig, branchName string, minChecks int) PullRequestStatus {
	/*
		Turn each type into a CSV with initial key field.
		gh pr view 73 --json "state,reviews,statusCheckRollup" --jq '(.reviews[] | select(.state == "APPROVED") | "approver," + .author.login + "," + .state+","+.commit.oid),(.statusCheckRollup[] | "check," + .status + ","+.conclusion+","+.state),("state," + .state)'
		approved,jallentest1
		check,COMPLETED,SUCCESS,SUCCESS
		state,OPEN
	*/
	if minChecks == -1 {
		minChecks = getMinChecks(appConfig)
	}
	lastCommit := GetBranchLatestCommit(branchName)
	jq := "(.reviews[] | select(.state == \"APPROVED\" and .commit.oid == \"" + lastCommit + "\") | \"approver,\" + .author.login)," +
		"(.statusCheckRollup[] | \"check,\" + .status + \",\"+.conclusion+\",\"+.state)," +
		"(\"state,\" + .state)"
	out := ExecuteOrDie(ExecuteOptions{Retries: GhRetries},
		"gh", "pr", "view", branchName, "--json", "state,reviews,statusCheckRollup", "--jq", jq)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	status := PullRequestStatus{Checks: PullRequestChecksStatus{MinChecks: minChecks}, Approvers: []string{}, State: PullRequestStateClosed}
	for _, line := range lines {
		fields := strings.Split(line, ",")
		if len(fields) > 0 {
			switch fields[0] {
			case "approver":
				status.Approvers = append(status.Approvers, fields[1])
			case "check":
				updatePullRequestChecksStatus(&status.Checks, fields[1], fields[2], fields[3])
			case "state":
				switch fields[1] {
				case "MERGED":
					status.State = PullRequestStateMerged
				case "OPEN":
					status.State = PullRequestStateOpen
				default:
					status.State = PullRequestStateClosed
				}
			default:
				panic("Unexpected key " + fields[0])
			}
		}
	}
	slices.Sort(status.Approvers)
	status.Approvers = slices.Compact(status.Approvers)
	return status
}
