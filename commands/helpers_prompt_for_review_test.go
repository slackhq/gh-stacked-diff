package commands

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

type promptForReviewTestCase struct {
	name          string
	configValue   string
	expectPrompt  bool
	expectPrReady bool
}

var promptForReviewTestCases = []promptForReviewTestCase{
	{"Never_NoPromptShown", "never", false, false},
	{"PromptN_DefaultsToNo", "promptN", true, false},
	{"PromptY_DefaultsToYes", "promptY", true, true},
}

// runPromptForReviewTest sets up interactive programs and assertions for a promptForReview
// config test. setupCommits is called to set up the git state before the test runs and must
// return the command args (without --config) to execute.
func runPromptForReviewTest(t *testing.T, tt promptForReviewTestCase, setupCommits func(testExecutor *util.TestExecutor) []string) {
	t.Helper()
	testExecutor := testutil.InitTest(t, slog.LevelError)

	commandArgs := setupCommits(testExecutor)

	if tt.expectPrReady {
		testExecutor.SetResponse(
			strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
			nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)
	}

	programIndex := 0
	// First interactive prompt (commit/PR selection)
	interactive.SendToProgram(programIndex, interactive.NewMessageKey(tea.KeyEnter))
	programIndex++
	if tt.expectPrompt {
		// Mark PR as ready for review when checks pass?
		interactive.SendToProgram(programIndex, interactive.NewMessageKey(tea.KeyEnter))
		programIndex++
	}
	if tt.expectPrReady {
		// Reviewers to add when checks pass?
		interactive.SendToProgram(programIndex, interactive.NewMessageKey(tea.KeyEnter))
	}

	args := []string{"--config", "promptForReview=" + tt.configValue}
	args = append(args, commandArgs...)
	if tt.expectPrReady {
		args = append(args, "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks))
	}
	testParseArguments(args...)

	assertGhSubcommandCalled(t, testExecutor.Responses, tt.expectPrReady, "pr", "ready")
}

// assertGhSubcommandCalled asserts that a gh command with the given arg prefix
// was (or was not) called.
func assertGhSubcommandCalled(t *testing.T, responses []util.ExecutedResponse, expected bool, args ...string) {
	t.Helper()
	called := slices.ContainsFunc(responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh" && len(next.Args) >= len(args) &&
			slices.Equal(next.Args[:len(args)], args)
	})
	assert.Equal(t, expected, called, "gh %s: %v", strings.Join(args, " "),
		util.FilterSlice(responses, func(next util.ExecutedResponse) bool {
			return next.ProgramName == "gh"
		}))
}
