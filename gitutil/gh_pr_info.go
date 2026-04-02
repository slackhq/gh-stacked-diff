package gitutil

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

var GhDelim = "|stackeddiff-delim|"

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
	slog.Debug(fmt.Sprintf("Checking for merged PR with head branch: %s", branchName))
	output := util.ExecuteOrDie(
		util.ExecuteOptions{Retries: GhRetries},
		"gh", "pr", "list",
		"--head", branchName,
		"--state", "merged",
		"--json", "number,title,state",
		"--jq", ".[0] | \"\\(.number)|\\(.title)|\\(.state)\"",
	)

	output = strings.TrimSpace(output)
	slog.Debug(fmt.Sprintf("gh pr list output for branch %s: '%s'", branchName, output))

	if output == "" || output == "null" || output == "||" || output == "null|null|null" {
		slog.Debug(fmt.Sprintf("No merged PR found for branch %s", branchName))
		return nil
	}

	// Parse the output: "number|title|state"
	parts := strings.SplitN(output, "|", 3)
	if len(parts) != 3 {
		slog.Warn(fmt.Sprintf("Unexpected PR output format for branch %s: %s", branchName, output))
		return nil
	}

	// Check if any part is empty or "null" (which means the PR data was incomplete/missing)
	if parts[0] == "" || parts[0] == "null" || parts[1] == "" || parts[1] == "null" || parts[2] == "" || parts[2] == "null" {
		slog.Debug(fmt.Sprintf("No merged PR found for branch %s (empty or null PR data)", branchName))
		return nil
	}

	slog.Debug(fmt.Sprintf("Found merged PR for branch %s: #%s - %s (%s)", branchName, parts[0], parts[1], parts[2]))

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
	output := util.ExecuteOrDie(
		util.ExecuteOptions{Retries: GhRetries},
		"gh", "pr", "list",
		"--head", branchName,
		"--state", "open",
		"--json", "number,title,state,body",
		"--jq", ".[0] | \"\\(.number)"+GhDelim+"\\(.title)"+GhDelim+"\\(.state)"+GhDelim+"\\(.body)\"",
	)

	// Parse the output: "number|title|state|body"
	// Check if status is not present (which means the PR data was incomplete/missing)
	parts := strings.SplitN(output, GhDelim, 5)
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
