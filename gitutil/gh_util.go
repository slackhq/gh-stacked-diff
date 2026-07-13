package gitutil

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

type PullRequestState int

const (
	PullRequestStateClosed PullRequestState = iota
	PullRequestStateOpen
	PullRequestStateMerged
)

const (
	ReviewStateApproved         = "APPROVED"
	ReviewStateChangesRequested = "CHANGES_REQUESTED"
	ReviewStateCommented        = "COMMENTED"
)

type LatestReview struct {
	Login        string
	State        string
	BodyLength   int
	CommentCount int
}

// minMeaningfulCommentLength filters out short auto-generated or boilerplate review bodies.
const minMeaningfulCommentLength = 10

func (r LatestReview) HasComments() bool {
	return r.BodyLength > minMeaningfulCommentLength || r.CommentCount > 0
}

type PullRequestStatus struct {
	Checks         PullRequestChecksStatus
	State          PullRequestState
	TotalReviewers int
	LatestReviews  []LatestReview
	CanMerge       bool
	IsDraft        bool
	IsInMergeQueue bool
}

/*
Returns users that have already approved the latest commit.
This uses the "reviews" field filtered by commit OID, unlike GetPullRequestStatus
which uses "latestReviews" (the latest review per user, regardless of commit).

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
	util.RequireHexString(lastCommit)
	jq := ".reviews[] | select(.state == \"APPROVED\" and .commit.oid == \"" + lastCommit + "\") | .author.login"
	out := util.ExecuteOrDie(util.ExecuteOptions{},
		"gh", "pr", "view", branchName, "--json", "reviews", "--jq", jq, GhRepoArgs())
	approvingUsers := strings.Fields(out)
	slices.Sort(approvingUsers)
	return slices.Compact(approvingUsers)
}

// Returns full commit hash of branch with name of branchName, or "" if no such branch.
func GetBranchLatestCommit(branchName string) string {
	out, err := util.Execute(util.ExecuteOptions{}, "git", "log", "-n", "1", "--pretty=format:%H", branchName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

/*
GetPullRequestStatus fetches PR status via a single gh api graphql call using a
jq query that produces CSV lines. Example output:

	rateLimit,1,4999,5000,2025-01-01T00:00:00Z
	check,COMPLETED,SUCCESS,
	check,,,SUCCESS
	state,OPEN
	reviewRequestCount,3
	latestReview,someuser,APPROVED,4,0
	latestReview,otheruser,CHANGES_REQUESTED,0,2
	mergeStateStatus,CLEAN
	isDraft,false
	autoMerge,false
*/
func GetPullRequestStatus(branchName string, minChecks int) PullRequestStatus {
	if minChecks == -1 {
		minChecks = getMinChecks()
	}
	nameWithOwner := GetRepoNameWithOwner()
	parts := strings.SplitN(nameWithOwner, "/", 2)
	if len(parts) != 2 {
		panic("could not parse repo name with owner: " + nameWithOwner)
	}
	query := `query($owner:String!,$repo:String!,$prHeadRef:String!){` +
		`repository(owner:$owner,name:$repo){` +
		`pullRequests(headRefName:$prHeadRef,states:[OPEN,CLOSED,MERGED],first:1){nodes{` +
		`state isDraft mergeStateStatus mergeQueueEntry{id} autoMergeRequest{enabledAt}` +
		`reviewRequests{totalCount}` +
		`latestReviews(last:100){nodes{author{login} state body comments{totalCount}}}` +
		`commits(last:1){nodes{commit{statusCheckRollup{contexts(last:100){nodes{` +
		`__typename ... on CheckRun{status conclusion} ... on StatusContext{state}` +
		`}}}}}}` +
		`}}}` +
		`rateLimit{limit cost remaining resetAt}` +
		`}`
	jq := "def pr: .data.repository.pullRequests.nodes[0];" +
		`("rateLimit," + (.data.rateLimit.cost|tostring) + "," + (.data.rateLimit.remaining|tostring) + "," + (.data.rateLimit.limit|tostring) + "," + .data.rateLimit.resetAt),` +
		`(pr.commits.nodes[0].commit.statusCheckRollup.contexts.nodes[]? | if .__typename=="CheckRun" then "check,"+.status+","+(.conclusion//"")+","  else "check,,,"+(.state//"") end),` +
		`("state," + pr.state),` +
		`("reviewRequestCount," + (pr.reviewRequests.totalCount|tostring)),` +
		`(pr.latestReviews.nodes[]? | "latestReview,"+.author.login+","+.state+","+(.body|length|tostring)+","+(.comments.totalCount|tostring)),` +
		`("mergeStateStatus," + (pr.mergeStateStatus // "")),` +
		`("isDraft," + (if pr.isDraft then "true" else "false" end)),` +
		`("autoMerge," + (if (pr.autoMergeRequest != null) or (pr.mergeQueueEntry != null) then "true" else "false" end))`
	out := util.ExecuteOrDie(util.ExecuteOptions{},
		"gh", "api", "graphql",
		"-f", "query="+query,
		"-f", "owner="+parts[0],
		"-f", "repo="+parts[1],
		"-f", "prHeadRef="+branchName,
		"--jq", jq)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	status := PullRequestStatus{Checks: PullRequestChecksStatus{MinChecks: minChecks}}
	for _, line := range lines {
		fields := strings.Split(line, ",")
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "rateLimit":
			if len(fields) >= 5 {
				slog.Debug(fmt.Sprint("GetPullRequestStatus rateLimit: cost=", fields[1], " remaining=", fields[2], " limit=", fields[3], " resetAt=", fields[4]))
			}
		case "check":
			if len(fields) >= 4 {
				updatePullRequestChecksStatus(&status.Checks, fields[1], fields[2], fields[3])
			} else {
				slog.Warn(fmt.Sprint("malformed check line in graphql output: ", line))
			}
		case "state":
			switch fields[1] {
			case "MERGED":
				status.State = PullRequestStateMerged
			case "OPEN":
				status.State = PullRequestStateOpen
			case "CLOSED":
				status.State = PullRequestStateClosed
			}
		case "reviewRequestCount":
			count, err := strconv.Atoi(fields[1])
			if err == nil {
				status.TotalReviewers += count
			}
		case "latestReview":
			if len(fields) >= 4 {
				status.TotalReviewers++
				bodyLen, _ := strconv.Atoi(fields[3])
				commentCount := 0
				if len(fields) > 4 {
					commentCount, _ = strconv.Atoi(fields[4])
				}
				status.LatestReviews = append(status.LatestReviews, LatestReview{
					Login:        fields[1],
					State:        fields[2],
					BodyLength:   bodyLen,
					CommentCount: commentCount,
				})
			} else {
				slog.Warn(fmt.Sprint("malformed latestReview line in graphql output: ", line))
			}
		case "mergeStateStatus":
			status.CanMerge = fields[1] == "CLEAN"
		case "isDraft":
			status.IsDraft = fields[1] == "true"
		case "autoMerge":
			if fields[1] == "true" {
				status.IsInMergeQueue = true
			}
		default:
			slog.Warn(fmt.Sprint("unexpected key in graphql output: ", fields[0]))
		}
	}
	return status
}
