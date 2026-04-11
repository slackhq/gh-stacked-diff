package gitutil

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

const GhDelim = "|stackeddiff-delim|"

// PrInfo holds information about a pull request
type PrInfo struct {
	Number string
	Title  string
	State  string
	Body   string
}

// GetMergedPR checks if a branch has a merged pull request.
// Returns the PR information if found, nil otherwise.
func GetMergedPR(branchName string) *PrInfo {
	// Check for merged PRs with this branch as the head
	slog.Debug(fmt.Sprint("Checking for merged PR with head branch: ", branchName))
	output := util.ExecuteOrDieTrimmed(
		util.ExecuteOptions{Retries: GhRetries},
		"gh", "pr", "list",
		"--head", branchName,
		"--state", "merged",
		"--json", "number,title,state",
		"--jq", ".[0] | \"\\(.number)"+GhDelim+"\\(.title)"+GhDelim+"\\(.state)\"",
	)

	slog.Debug(fmt.Sprint("gh pr list output for branch ", branchName, ": '", output, "'"))

	nullOutput := GhDelim + GhDelim
	nullNullOutput := "null" + GhDelim + "null" + GhDelim + "null"
	if output == "" || output == "null" || output == nullOutput || output == nullNullOutput {
		slog.Debug(fmt.Sprint("No merged PR found for branch ", branchName))
		return nil
	}

	// Parse the output: "number<delim>title<delim>state"
	parts := strings.SplitN(output, GhDelim, 3)
	if len(parts) != 3 {
		slog.Warn(fmt.Sprint("Unexpected PR output format for branch ", branchName, ": ", output))
		return nil
	}

	// Check if any part is empty or "null" (which means the PR data was incomplete/missing)
	if parts[0] == "" || parts[0] == "null" || parts[1] == "" || parts[1] == "null" || parts[2] == "" || parts[2] == "null" {
		slog.Debug(fmt.Sprint("No merged PR found for branch ", branchName, " (empty or null PR data)"))
		return nil
	}

	slog.Debug(fmt.Sprint("Found merged PR for branch ", branchName, ": #", parts[0], " - ", parts[1], " (", parts[2], ")"))

	return &PrInfo{
		Number: parts[0],
		Title:  parts[1],
		State:  parts[2],
	}
}

// GetUnmergedPR checks if a branch has an open (unmerged) pull request.
// Returns the PR information if found, nil otherwise.
func GetUnmergedPR(branchName string) *PrInfo {
	// Check for open PRs with this branch as the head
	output := util.ExecuteOrDieTrimmed(
		util.ExecuteOptions{Retries: GhRetries},
		"gh", "pr", "list",
		"--head", branchName,
		"--state", "open",
		"--json", "number,title,state,body",
		"--jq", ".[0] | \"\\(.number)"+GhDelim+"\\(.title)"+GhDelim+"\\(.state)"+GhDelim+"\\(.body)\"",
	)

	// Parse the output: "number|title|state|body"
	// Check if status is not present (which means the PR data was incomplete/missing)
	parts := strings.SplitN(output, GhDelim, 4)
	if len(parts) != 4 || strings.ToUpper(parts[2]) != "OPEN" {
		return nil
	}

	return &PrInfo{
		Number: parts[0],
		Title:  parts[1],
		State:  parts[2],
		Body:   parts[3],
	}
}
