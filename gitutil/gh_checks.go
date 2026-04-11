package gitutil

import (
	"bufio"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

const DefaultMinChecks = 1
const MaxChecks = 5

// Cached value from minChecks api call if there were checks run.
var minChecksHistory = util.NewHistoricalData("min-checks.history", 2)
var minChecksCacheDuration = 48 * time.Hour

type PullRequestChecksStatus struct {
	Pending   int
	Failing   int
	Passing   int
	MinChecks int
}

func (s PullRequestChecksStatus) PercentageComplete() float32 {
	if s.Total() == 0 || s.Total() < s.MinChecks {
		return 0
	}
	numRun := s.Passing + s.Failing
	return float32(numRun) / float32(s.Total())
}

func (s PullRequestChecksStatus) IsSuccess() bool {
	return s.Total() >= s.MinChecks && s.Passing > 0 && s.Failing == 0 && s.Pending == 0
}

func (s PullRequestChecksStatus) IsFailing() bool {
	return s.Failing > 0
}

func (s PullRequestChecksStatus) Total() int {
	return s.Failing + s.Passing + s.Pending
}

/*
 * Logic copied from https://github.com/cli/cli/blob/57fbe4f317ca7d0849eeeedb16c1abc21a81913b/api/queries_pr.go#L258-L274
 */
func GetChecksStatus(branchName string, minChecks int) PullRequestChecksStatus {
	if minChecks == -1 {
		minChecks = getMinChecks()
	}
	summary := PullRequestChecksStatus{MinChecks: minChecks}
	stateString := util.ExecuteOrDie(util.ExecuteOptions{Retries: GhRetries},
		"gh", "pr", "view", branchName, "--json", "statusCheckRollup",
		"--jq", ".statusCheckRollup[] | .status, .conclusion, .state")
	scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(stateString)))
	for scanner.Scan() {
		status := scanner.Text()
		scanner.Scan()
		conclusion := scanner.Text()
		scanner.Scan()
		state := scanner.Text()
		updatePullRequestChecksStatus(&summary, status, conclusion, state)
	}
	return summary
}

func updatePullRequestChecksStatus(checks *PullRequestChecksStatus, status string, conclusion string, state string) {
	if state == "" {
		if status == "COMPLETED" {
			state = conclusion
		} else {
			state = status
		}
	}
	switch state {
	case "SUCCESS", "NEUTRAL", "SKIPPED":
		checks.Passing++
	case "ERROR", "FAILURE", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED":
		checks.Failing++
	default: // "EXPECTED", "REQUESTED", "WAITING", "QUEUED", "PENDING", "IN_PROGRESS", "STALE"
		checks.Pending++
	}
}

func getMinChecks() int {
	cache.minChecksOnce.Do(func() {
		minChecksFromHistory := getMinChecksFromHistory()
		if minChecksFromHistory != -1 {
			cache.minChecks = minChecksFromHistory
			return
		}
		jq := ".[].statusCheckRollup | length"
		// Github sometimes returns an error for this command so retry and then fallback to default.
		out, err := util.Execute(util.ExecuteOptions{Retries: GhRetries},
			"gh", "pr", "list", "--state", "merged", "--base", GetRemoteMainBranchOrDie(),
			"--json", "statusCheckRollup", "--jq", jq)
		if err != nil {
			slog.Warn("Could not determine min checks so using default " + fmt.Sprint(DefaultMinChecks))
			cache.minChecks = DefaultMinChecks
			return
		}
		allNumChecks := util.MapSlice(strings.Fields(out), func(next string) int {
			numChecks, err := strconv.Atoi(next)
			if err != nil {
				panic(err)
			}
			return numChecks
		})
		if len(allNumChecks) == 0 {
			return
		}

		minChecks := slices.Min(allNumChecks)
		slog.Debug(fmt.Sprint("Checks from PRs are ", allNumChecks, " min is ", minChecks))
		cache.minChecks = min(minChecks, MaxChecks)
		setMinChecksToHistory(cache.minChecks)
	})
	return cache.minChecks
}

func getMinChecksFromHistory() int {
	history := minChecksHistory.ReadHistory()
	if len(history) == 2 {
		when, timeParseErr := time.Parse(time.RFC3339, history[0])
		if timeParseErr == nil {
			if time.Since(when) < minChecksCacheDuration {
				numChecks, intConvErr := strconv.Atoi(history[1])
				if intConvErr == nil {
					return numChecks
				}
			}
		}
	}
	return -1
}

func setMinChecksToHistory(minChecks int) {
	minChecksHistory.SetHistory(
		[]string{time.Now().Format(time.RFC3339), fmt.Sprint(minChecks)},
	)
}
