package commands

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	tea "github.com/charmbracelet/bubbletea"

	"errors"

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
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", util.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(util.DefaultMinChecks), "--reviewers=mybestie", allCommits[0].Commit)

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
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", util.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(util.DefaultMinChecks), "--indicator=list", "--reviewers=mybestie", "1")

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
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", util.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))
	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(util.DefaultMinChecks), "--indicator=list", "--reviewers=mybestie")

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
	checksSuccess := strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", util.DefaultMinChecks)
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

	out := testParseArguments("--log-level=info", "add-reviewers", "--min-checks", fmt.Sprint(util.DefaultMinChecks), "--reviewers=alreadyapproved2,mybestie,alreadyapproved1", "1")

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
	checksSuccess := strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", util.DefaultMinChecks)
	testExecutor.SetResponseFunc(checksSuccess, nil, func(programName string, args ...string) bool {
		return programName == "gh" &&
			args[0] == "pr" &&
			args[1] == "view" &&
			slices.Contains(args, "statusCheckRollup")
	})

	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(util.DefaultMinChecks), "--reviewers=mybestie", "1")

	// Clear responses.
	testExecutor.Responses = []util.ExecutedResponse{}
	// What reviewers?
	interactive.SendToProgram(0,
		// History
		interactive.NewMessageKey(tea.KeyUp),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(util.DefaultMinChecks), "1")

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
	checksSuccess := strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", util.DefaultMinChecks)
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
	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(util.DefaultMinChecks), "1")

	// Clear responses.
	testExecutor.Responses = []util.ExecutedResponse{}
	// What reviewers?
	interactive.SendToProgram(1,
		// History
		interactive.NewMessageKey(tea.KeyUp),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(util.DefaultMinChecks), "1")

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "edit", allCommits[0].Branch, "--add-reviewer", "my"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}

func TestSdAddReviewers_WhenMergeFlag_EnablesAutoMerge(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", util.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	testParseArguments("add-reviewers", "--min-checks", fmt.Sprint(util.DefaultMinChecks), "--merge", "--reviewers=mybestie", allCommits[0].Commit)

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "merge", allCommits[0].Branch, "--auto", "--squash"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}

func TestSdAddReviewers_WhenMinChecksFails_UsesDefault(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
	checksSuccess := strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", util.DefaultMinChecks)
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
