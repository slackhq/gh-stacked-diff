package commands

import (
	"bytes"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	tea "github.com/charmbracelet/bubbletea"

	"errors"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func TestSdAddReviewers_AddReviewers(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "--reviewers=mybestie", allCommits[0].Commit)

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "edit", allCommits[0].Branch, "--add-reviewer", "mybestie"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}

func TestSdAddReviewers_WhenUsingListIndicator_AddReviewers(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "--indicator=list", "--reviewers=mybestie", "1")

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "edit", allCommits[0].Branch, "--add-reviewer", "mybestie"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}

func TestSdAddReviewers_WhenOmittingCommitIndicator_AsksForSelection(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))
	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "--indicator=list", "--reviewers=mybestie")

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "edit", allCommits[0].Branch, "--add-reviewer", "mybestie"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}

func TestSdAddReviewers_WhenUserAlreadyApproved_DoesNotRequestReview(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
	checksSuccess := strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks)
	testExecutor.SetResponseFunc(checksSuccess, nil, func(programName string, args ...string) bool {
		return programName == "gh" &&
			args[0] == "pr" &&
			args[1] == "view" &&
			slices.Contains(args, "statusCheckRollup")
	})

	approvedUsers := "alreadyapproved1\nalreadyapproved2"
	testExecutor.SetResponseFunc(approvedUsers, nil, func(programName string, args ...string) bool {
		return programName == "gh" &&
			args[0] == "pr" &&
			args[1] == "view" &&
			slices.Contains(args, "reviews")
	})

	out := testParseArguments("--log-level=info", "add-reviewers", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "--reviewers=alreadyapproved2,mybestie,alreadyapproved1", "1")

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "edit", allCommits[0].Branch, "--add-reviewer", "mybestie"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))

	assert.Contains(out, "alreadyapproved2,alreadyapproved1")
}

func TestSdAddReviewers_UserChoosesHistory_ChoosesSameReviewers(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
	checksSuccess := strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks)
	testExecutor.SetResponseFunc(checksSuccess, nil, func(programName string, args ...string) bool {
		return programName == "gh" &&
			args[0] == "pr" &&
			args[1] == "view" &&
			slices.Contains(args, "statusCheckRollup")
	})

	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "--reviewers=mybestie", "1")

	// Clear responses.
	testExecutor.Responses = []util.ExecutedResponse{}
	// What reviewers?
	interactive.SendToProgram(0,
		// History
		interactive.NewMessageKey(tea.KeyUp),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "1")

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "edit", allCommits[0].Branch, "--add-reviewer", "mybestie"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}

func TestSdAddReviewers_UserChoosesHistoryFromTyped_ChoosesSameReviewers(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
	checksSuccess := strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks)
	testExecutor.SetResponseFunc(checksSuccess, nil, func(programName string, args ...string) bool {
		return programName == "gh" &&
			args[0] == "pr" &&
			args[1] == "view" &&
			slices.Contains(args, "statusCheckRollup")
	})

	// What reviewers?
	interactive.SendToProgram(0,
		interactive.NewMessageRune('m'),
		interactive.NewMessageRune('y'),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "1")

	// Clear responses.
	testExecutor.Responses = []util.ExecutedResponse{}
	// What reviewers?
	interactive.SendToProgram(1,
		// History
		interactive.NewMessageKey(tea.KeyUp),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "1")

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "edit", allCommits[0].Branch, "--add-reviewer", "my"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}

func TestSdAddReviewers_WhenNoReviewersSelected_DoesNotAddReviewers(t *testing.T) {
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testExecutor.SetResponse(
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	// Select PR, then enter empty reviewers.
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))
	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "1")

	assertGhSubcommandCalled(t, testExecutor.Responses, false, "pr", "edit")
	assertGhSubcommandCalled(t, testExecutor.Responses, true, "pr", "ready")
}

func TestSdAddReviewers_WhenMergeFlag_EnablesAutoMerge(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "--merge", "--reviewers=mybestie", allCommits[0].Commit)

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "merge", allCommits[0].Branch, "--auto", "--squash"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}

func TestSdAddReviewers_WhenChecksFail_ShowsErrorInsteadOfStackTrace(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	// Return failing checks: one SUCCESS and one FAILURE to meet min checks.
	failingChecks := "SUCCESS\nSUCCESS\nSUCCESS\nCOMPLETED\nFAILURE\n\n"
	testExecutor.SetResponse(
		failingChecks,
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	// The panic from checks failing should be caught by SendErrorOnPanic and
	// propagated through the bubbletea program, then re-panicked in runProgram,
	// and finally caught by ExecuteCommand's recover which prints the error and
	// calls appConfig.Exit(1). In tests, Exit panics, so we recover it here.
	// If SendErrorOnPanic fails to recover (the bug), the goroutine panic
	// crashes the entire test process instead of being caught here.
	out := new(bytes.Buffer)
	assert.Panics(func() {
		testParseArgumentsWithOut(out, "add-reviewers", "--min-checks", "2", "--reviewers=mybestie", "1")
	})

	assert.Contains(out.String(), "Checks failed")
}

func TestSdAddReviewers_WhenMinChecksFails_UsesDefault(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
	checksSuccess := strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks)
	testExecutor.SetResponseFunc(checksSuccess, nil, func(programName string, args ...string) bool {
		return programName == "gh" &&
			args[0] == "pr" &&
			args[1] == "view" &&
			slices.Contains(args, "statusCheckRollup")
	})

	testExecutor.SetResponse("", errors.New("error"), "gh", "pr", "list", "--state", "merged", util.MatchAnyRemainingArgs)

	testParseArguments("add-reviewers", "--reviewers=mybestie", "1")

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "edit", allCommits[0].Branch, "--add-reviewer", "mybestie"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}
