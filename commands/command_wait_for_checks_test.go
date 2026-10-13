package commands

import (
	"bytes"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func TestSdWaitForChecks_WhenChecksPass_DoesNotMarkReadyOrAddReviewers(t *testing.T) {
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	testExecutor.SetResponse(
		prStatusWithChecks(gitutil.DefaultMinChecks),
		nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	testParseArguments("wait-for-checks", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), allCommits[0].Commit)

	assertGhSubcommandCalled(t, testExecutor.Responses, false, "pr", "ready")
	assertGhSubcommandCalled(t, testExecutor.Responses, false, "pr", "edit")
	assertGhSubcommandCalled(t, testExecutor.Responses, false, "pr", "merge")
}

func TestSdWaitForChecks_WhenChecksFail_Panics(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	failingChecks := "check,COMPLETED,SUCCESS,\ncheck,COMPLETED,FAILURE,\nstate,OPEN\nnumber,1\nreviewRequestCount,0\nmergeStateStatus,BLOCKED\nisDraft,false\nautoMerge,false"
	testExecutor.SetResponse(failingChecks, nil, "gh", "api", "graphql", util.MatchAnyRemainingArgs)

	out := new(bytes.Buffer)
	assert.Panics(func() {
		testParseArgumentsWithOut(out, "wait-for-checks", "--min-checks", "2", "1")
	})
	assert.Contains(out.String(), "Checks failed")
}
