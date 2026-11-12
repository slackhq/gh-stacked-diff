package gitutil

import (
	"os/exec"
	"strings"
	"testing"
)

func TestPullRequestStatusJq_WhenDuplicateCheckRuns_KeepsOnlyLatest(t *testing.T) {
	jqPath, err := exec.LookPath("jq")
	if err != nil {
		t.Skip("jq not installed")
	}
	response := `{"data":{"repository":{"pullRequests":{"nodes":[{
		"state":"OPEN","isDraft":false,"mergeStateStatus":"BLOCKED",
		"reviewRequests":{"totalCount":0},"latestReviews":{"nodes":[]},
		"commits":{"nodes":[{"commit":{"oid":"abc","statusCheckRollup":{"contexts":{"nodes":[
			{"__typename":"CheckRun","name":"approve","status":"COMPLETED","conclusion":"CANCELLED","startedAt":"2026-11-11T19:25:08Z","checkSuite":{"workflowRun":{"workflow":{"name":"Auto-approve"}}}},
			{"__typename":"CheckRun","name":"approve","status":"COMPLETED","conclusion":"SUCCESS","startedAt":"2026-11-11T19:25:11Z","checkSuite":{"workflowRun":{"workflow":{"name":"Auto-approve"}}}},
			{"__typename":"CheckRun","name":"approve","status":"COMPLETED","conclusion":"FAILURE","startedAt":"2026-11-11T19:00:00Z","checkSuite":{"workflowRun":{"workflow":{"name":"Other workflow"}}}},
			{"__typename":"StatusContext","state":"PENDING"}
		]}}}}]}}]}}}}`

	cmd := exec.Command(jqPath, "-r", pullRequestStatusJq())
	cmd.Stdin = strings.NewReader(response)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("jq failed: %v\n%s", err, out)
	}

	var checkLines []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "check,") {
			checkLines = append(checkLines, line)
		}
	}
	got := strings.Join(checkLines, "|")
	// Same name in a different workflow is a separate check, so its failure is kept.
	want := "check,COMPLETED,SUCCESS,|check,COMPLETED,FAILURE,|check,,,PENDING"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
