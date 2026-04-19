package commands

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"testing"

	"github.com/stretchr/testify/assert"

	"errors"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSdNew_OnRepoWithPreviousCommit_CreatesPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	testutil.AddCommit("second", "")
	allCommits := templates.GetNewCommits("HEAD", "")

	testParseArguments("new", "1")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", allCommits[0].Branch)
	commitsOnNewBranch := templates.GetNewCommits("HEAD", "")
	assert.Equal(1, len(commitsOnNewBranch))
	assert.Equal(allCommits[0].Subject, commitsOnNewBranch[0].Subject)
}

func TestSdNew_WithMiddleCommit_CreatesPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	testutil.AddCommit("second", "")

	testutil.AddCommit("third", "")
	allCommits := templates.GetNewCommits("HEAD", "")

	testParseArguments("new", "1")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", allCommits[0].Branch)
	commitsOnNewBranch := templates.GetNewCommits("HEAD", "")
	assert.Equal(1, len(commitsOnNewBranch))
	assert.Equal(allCommits[0].Subject, commitsOnNewBranch[0].Subject)
}

func TestSdNew_CreatesPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	out := testParseArguments("log")

	assert.Contains(out, "✓")
}

func TestSdNew_WithReviewers_AddReviewers(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	testParseArguments("new", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "--reviewers=mybestie", "1")

	allCommits := templates.GetAllCommits()

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "edit", allCommits[0].Branch, "--add-reviewer", "mybestie"}
		return next.ProgramName == "gh" && len(next.Args) >= len(ghExpectedArgs) && slices.Equal(next.Args[:len(ghExpectedArgs)], ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}

func TestSdNew_WithReviewersFlag_SavesReviewersToHistory(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testExecutor.SetResponse(
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	// Verify no history before running.
	assert.Empty(interactive.ReviewersHistory.ReadHistory())

	testParseArguments("new", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "--reviewers=mybestie", "1")

	history := interactive.ReviewersHistory.ReadHistory()
	assert.Contains(history, "mybestie", "reviewers passed via --reviewers flag should be saved to history")
}

func TestSdNew_WhenUsingListIndex_UsesCorrectList(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")
	testutil.AddCommit("fourth", "")

	allCommits := templates.GetAllCommits()

	testParseArguments("new", "2")

	assert.Equal(true, gitutil.RemoteHasBranch(allCommits[1].Branch))
}

func TestSdNew_WhenDraftNotSupported_TriesAgainWithoutDraft(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	draftNotSupported := "pull request create failed: GraphQL: Draft pull requests are not supported in this repository. (createPullRequest)"
	testExecutor.SetResponseFunc(draftNotSupported, errors.New("Exit code 1"), func(programName string, args ...string) bool {
		return programName == "gh" && args[0] == "pr" && args[1] == "create" && slices.Contains(args, "--draft")
	})

	out := testParseArguments("--log-level=info", "new", "1")

	assert.Contains(out, "Use \"--draft=false\" to avoid this warning")
	assert.Contains(out, "Created PR ")
}

func TestSdNew_WhenTwoPrsOnRoot_CreatesFromRoot(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")

	testParseArguments("new", "2")
	testParseArguments("new", "1")

	mainCommits := templates.GetAllCommits()

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", mainCommits[1].Branch)
	firstCommits := templates.GetAllCommits()
	assert.Equal(2, len(firstCommits))
	assert.Equal(mainCommits[1].Subject, firstCommits[0].Subject)
	assert.Equal(testutil.InitialCommitSubject, firstCommits[1].Subject)

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", mainCommits[0].Branch)
	secondCommits := templates.GetAllCommits()
	assert.Equal(2, len(secondCommits))
	assert.Equal(mainCommits[0].Subject, secondCommits[0].Subject)
	assert.Equal(testutil.InitialCommitSubject, firstCommits[1].Subject)
}

func TestSdNew_WhenCherryPickFails_RestoresBranch(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testutil.CommitFileChange("second", "first", "changes")

	allCommits := templates.GetAllCommits()

	restoreBranch := util.GetCurrentBranchName()
	defer func() {
		r := recover()
		if r != nil {
			assert.Equal(restoreBranch, util.GetCurrentBranchName())
			assert.Equal(allCommits, templates.GetAllCommits())
		}
	}()

	testParseArguments("new", "1")

	assert.Fail("did not panic on conflicts with cherry-pick")
}

func TestSdNew_WhenNewPrFails_RestoresBranch(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	allCommits := templates.GetAllCommits()

	testExecutor.SetResponse("", errors.New("Exit Code 1"), "gh", "pr", "create", util.MatchAnyRemainingArgs)

	restoreBranch := util.GetCurrentBranchName()
	defer func() {
		r := recover()
		if r != nil {
			assert.Equal(restoreBranch, util.GetCurrentBranchName())
			assert.Equal(allCommits, templates.GetAllCommits())
		}
	}()

	testParseArguments("new", "1")

	assert.Fail("did not panic on PR create")
}

func TestSdNew_WhenDestinationCommitNotSpecified_CreatesPrWithSelectedCommit(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")

	interactive.SendToProgram(0,
		// What commit do you want to create a PR from?
		interactive.NewMessageKey(tea.KeyDown),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	interactive.SendToProgram(1,
		// Mark PR as ready for review when checks pass?
		interactive.NewMessageRune('n'),
	)
	testParseArguments("new")

	allCommits := templates.GetAllCommits()

	assert.True(gitutil.RemoteHasBranch(allCommits[1].Branch))
}

func TestSdNew_WhenDestinationCommitNotSpecified_WrapsCursorUp(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")

	interactive.SendToProgram(0,
		// What commit do you want to create a PR from?
		interactive.NewMessageKey(tea.KeyUp),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	interactive.SendToProgram(1,
		// Mark PR as ready for review when checks pass?
		interactive.NewMessageRune('n'),
	)
	testParseArguments("new")

	allCommits := templates.GetAllCommits()

	assert.True(gitutil.RemoteHasBranch(allCommits[2].Branch))
}

func TestSdNew_WhenDestinationCommitNotSpecified_SkipsDisabledRows(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")

	// Create a PR for "second" (index 2), making it disabled for "sd new".
	testParseArguments("new", "2")

	interactive.SendToProgram(0,
		// What commit do you want to create a PR from?
		// Cursor starts on "first" (index 1, first enabled row).
		// Down should skip "second" (disabled) and land on "third".
		interactive.NewMessageKey(tea.KeyDown),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	interactive.SendToProgram(1,
		// Mark PR as ready for review when checks pass?
		interactive.NewMessageRune('n'),
	)
	testParseArguments("new")

	allCommits := templates.GetAllCommits()

	// "third" is at index 2 (0-indexed) in allCommits.
	assert.True(gitutil.RemoteHasBranch(allCommits[2].Branch))
}

func TestSdNew_WhenDestinationCommitNotSpecified_WrapsUpSkipsDisabledLastRow(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")

	// Create a PR for "third" (index 3), making the last row disabled for "sd new".
	testParseArguments("new", "3")

	interactive.SendToProgram(0,
		// What commit do you want to create a PR from?
		// Cursor starts on "first" (first enabled row).
		// Up should wrap, skip disabled "third", and land on "second".
		interactive.NewMessageKey(tea.KeyUp),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	interactive.SendToProgram(1,
		// Mark PR as ready for review when checks pass?
		interactive.NewMessageRune('n'),
	)
	testParseArguments("new")

	allCommits := templates.GetAllCommits()

	// "second" is at index 1 (0-indexed) in allCommits.
	assert.True(gitutil.RemoteHasBranch(allCommits[1].Branch))
}

func TestSdNew_WhenDestinationCommitNotSpecified_WrapsCursorDown(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")

	interactive.SendToProgram(0,
		// What commit do you want to create a PR from?
		interactive.NewMessageKey(tea.KeyDown),
		interactive.NewMessageKey(tea.KeyDown),
		interactive.NewMessageKey(tea.KeyDown),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	interactive.SendToProgram(1,
		// Mark PR as ready for review when checks pass?
		interactive.NewMessageRune('n'),
	)
	testParseArguments("new")

	allCommits := templates.GetAllCommits()

	assert.True(gitutil.RemoteHasBranch(allCommits[0].Branch))
}

func TestSdNew_WhenDestinationCommitNotSpecifiedAndManyCommits_PadsIndex(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")
	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")
	testutil.AddCommit("fourth", "")
	testutil.AddCommit("fifth", "")
	testutil.AddCommit("sixth", "")
	testutil.AddCommit("seventh", "")
	testutil.AddCommit("eighth", "")
	testutil.AddCommit("ninth", "")
	testutil.AddCommit("tenth", "")

	interactive.SendToProgram(0,
		// What commit do you want to create a PR from?
		interactive.NewMessageRune('q'),
	)
	out := new(bytes.Buffer)
	defer func() {
		r := recover()
		if r != nil {
			assert.Contains(out.String(), "│ 1│")
		}
	}()
	testParseArgumentsWithOut(out, "--log-level=error", "new")
	assert.Fail("did not panic on cancel")
}

func TestSdNew_WhenDestinationCommitNotSpecifiedAndManyCommitsAndExistingPr_PadsIndex(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")
	testParseArguments("new", "1")
	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")
	testutil.AddCommit("fourth", "")
	testutil.AddCommit("fifth", "")
	testutil.AddCommit("sixth", "")
	testutil.AddCommit("seventh", "")
	testutil.AddCommit("eighth", "")
	testutil.AddCommit("ninth", "")
	testutil.AddCommit("tenth", "")

	interactive.SendToProgram(0,
		// What commit do you want to create a PR from?
		interactive.NewMessageRune('q'),
	)
	out := new(bytes.Buffer)
	defer func() {
		r := recover()
		if r != nil {
			assert.Contains(out.String(), "│ 1  │")
			assert.Contains(out.String(), "│10 ✓│")
		}
	}()
	testParseArgumentsWithOut(out, "--log-level=error", "new")
	assert.Fail("did not panic on cancel")
}

func TestSdNew_WhenNoReviewersAndDraft_ConfirmReady_MarksPrReady(t *testing.T) {
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	interactive.SendToProgram(0,
		// What commit do you want to create a PR from?
		interactive.NewMessageKey(tea.KeyEnter),
	)
	interactive.SendToProgram(1,
		// Mark PR as ready for review when checks pass?
		interactive.NewMessageRune('y'),
	)
	interactive.SendToProgram(2,
		// Reviewers to add when checks pass?
		interactive.NewMessageKey(tea.KeyEnter),
	)
	testParseArguments("new", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks))

	allCommits := templates.GetAllCommits()

	// Verify gh pr ready was called
	assertGhSubcommandCalled(t, testExecutor.Responses, true, "pr", "ready", allCommits[0].Branch)

	// Verify no reviewers were added (no gh pr edit --add-reviewer)
	assertGhSubcommandCalled(t, testExecutor.Responses, false, "pr", "edit")
}

func TestSdNew_WhenNoReviewersAndDraft_EnterDefaultsToReady_MarksPrReady(t *testing.T) {
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	interactive.SendToProgram(0,
		// What commit do you want to create a PR from?
		interactive.NewMessageKey(tea.KeyEnter),
	)
	interactive.SendToProgram(1,
		// Mark PR as ready for review when checks pass? (Y/n) - Enter defaults to yes
		interactive.NewMessageKey(tea.KeyEnter),
	)
	interactive.SendToProgram(2,
		// Reviewers to add when checks pass?
		interactive.NewMessageKey(tea.KeyEnter),
	)
	testParseArguments("--config", "promptForReview=promptY", "new", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks))

	allCommits := templates.GetAllCommits()

	// Verify gh pr ready was called
	assertGhSubcommandCalled(t, testExecutor.Responses, true, "pr", "ready", allCommits[0].Branch)
}

func TestSdNew_WhenNoReviewersAndDraft_DeclineReady_PrStaysDraft(t *testing.T) {
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	interactive.SendToProgram(0,
		// What commit do you want to create a PR from?
		interactive.NewMessageKey(tea.KeyEnter),
	)
	interactive.SendToProgram(1,
		// Mark PR as ready for review when checks pass?
		interactive.NewMessageRune('n'),
	)
	testParseArguments("new")

	// Verify gh pr ready was NOT called
	assertGhSubcommandCalled(t, testExecutor.Responses, false, "pr", "ready")
}

func TestSdNew_ConfigPromptForReview(t *testing.T) {
	for _, tt := range promptForReviewTestCases {
		t.Run(tt.name, func(t *testing.T) {
			runPromptForReviewTest(t, tt, func(_ *util.TestExecutor) []string {
				testutil.AddCommit("first", "")
				return []string{"new"}
			})
		})
	}
}

func TestSdNew_WhenRemoteBranchUpdatedConcurrently_ForceWithLeaseRejectsThePush(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	allCommits := templates.GetAllCommits()
	branchName := allCommits[0].Branch

	// Simulate a concurrent user creating the same branch on origin.
	// Use a separate clone so our local repo has no remote-tracking ref for it.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "clone", "../remote-repo", "../concurrent-repo")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-C", "../concurrent-repo", "config", "user.email", "other@example.com")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-C", "../concurrent-repo", "config", "user.name", "Other User")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-C", "../concurrent-repo", "checkout", "-b", branchName)
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-C", "../concurrent-repo", "commit", "--allow-empty", "-m", "concurrent change")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-C", "../concurrent-repo", "push", "origin", branchName)

	// sd new creates the branch locally (--no-track), cherry-picks, and pushes.
	// --force-with-lease expects the remote branch to not exist (no tracking ref),
	// but the concurrent push created it, so the push is rejected.
	allCommitsBeforeNew := templates.GetAllCommits()
	defer func() {
		r := recover()
		if r != nil {
			assert.Equal(gitutil.GetLocalMainBranchOrDie(), util.GetCurrentBranchName())
			assert.Equal(allCommitsBeforeNew, templates.GetAllCommits())
		}
	}()

	testParseArguments("new", "1")

	assert.Fail("Expected push to fail due to --force-with-lease rejection")
}

func TestSdNew_WithTicketUrlPattern_ReplacesTicketNumberInPrDescription(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("CONV-1234 Add new feature", "")

	testParseArguments("--config", "ticketUrlPattern=https://jira.mycompany.com/browse/{TicketNumber}/description", "new", "1")

	// Find the gh pr create call and verify the --body contains the resolved ticket URL.
	prCreateCall, found := findGhPrCreateCall(testExecutor.Responses)
	assert.True(found, "expected gh pr create to be called")
	bodyIndex := slices.Index(prCreateCall.Args, "--body")
	assert.Greater(bodyIndex, -1, "expected --body flag in gh pr create args")
	body := prCreateCall.Args[bodyIndex+1]
	assert.Contains(body, "https://jira.mycompany.com/browse/CONV-1234/description", "ticket URL pattern should have {TicketNumber} replaced with actual ticket number")
	assert.Contains(body, "[CONV-1234]", "PR description should contain the ticket number as link text")
	assert.NotContains(body, "{TicketNumber}", "PR description should not contain unreplaced {TicketNumber} placeholder")
}

func TestSdNew_WithNoTemplate_UsesCommitBodyAsIs(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create a commit with markdown headers and backtick-wrapped content in the body.
	util.ExecuteOrDie(util.ExecuteOptions{}, "touch", "feature-file")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "-m", "Add feature", "-m", "Body paragraph\n\n#### Feature flag(s): `ANDROID_TEST_FLAG`")

	testParseArguments("new", "--no-template", "1")

	prCreateCall, found := findGhPrCreateCall(testExecutor.Responses)
	assert.True(found, "expected gh pr create to be called")

	titleIndex := slices.Index(prCreateCall.Args, "--title")
	assert.Greater(titleIndex, -1, "expected --title flag")
	title := prCreateCall.Args[titleIndex+1]
	assert.Equal("Add feature", title)

	bodyIndex := slices.Index(prCreateCall.Args, "--body")
	assert.Greater(bodyIndex, -1, "expected --body flag")
	body := prCreateCall.Args[bodyIndex+1]
	assert.Contains(body, "#### Feature flag(s):", "markdown headers should be preserved with --no-template")
	assert.Contains(body, "`ANDROID_TEST_FLAG`", "backtick-wrapped content should be preserved with --no-template")
}

func TestSdNew_WithNoTemplateAndTicketPrefix_SkipsTicketPrompt(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	// Create a commit with a ticket prefix. Without --no-template this would trigger
	// the interactive ticket URL pattern prompt.
	util.ExecuteOrDie(util.ExecuteOptions{}, "touch", "feature-file")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "add", ".")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "commit", "-m", "CONV-999 Add feature", "-m", "Body paragraph\n\n#### Ticket: [CONV-999](https://jira.tinyspeck.com/browse/CONV-999)\n\n#### Feature flag(s): `ANDROID_TEST_FLAG`")

	testParseArguments("new", "--no-template", "1")

	prCreateCall, found := findGhPrCreateCall(testExecutor.Responses)
	assert.True(found, "expected gh pr create to be called")

	titleIndex := slices.Index(prCreateCall.Args, "--title")
	assert.Greater(titleIndex, -1, "expected --title flag")
	title := prCreateCall.Args[titleIndex+1]
	assert.Equal("CONV-999 Add feature", title, "title should preserve the ticket prefix")

	bodyIndex := slices.Index(prCreateCall.Args, "--body")
	assert.Greater(bodyIndex, -1, "expected --body flag")
	body := prCreateCall.Args[bodyIndex+1]
	assert.Contains(body, "#### Ticket:", "ticket header should be preserved with --no-template")
	assert.Contains(body, "#### Feature flag(s):", "feature flag header should be preserved with --no-template")
	assert.Contains(body, "`ANDROID_TEST_FLAG`", "backtick-wrapped flag should be preserved with --no-template")
}

func findGhPrCreateCall(responses []util.ExecutedResponse) (util.ExecutedResponse, bool) {
	for _, r := range responses {
		if r.ProgramName == "gh" && len(r.Args) >= 2 && r.Args[0] == "pr" && r.Args[1] == "create" {
			return r, true
		}
	}
	return util.ExecutedResponse{}, false
}

func TestSdNew_WhenInSecondaryWorktree_UsesRemoteMainForBaseBranch(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetCurrentBranchName())

	mainPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Create a secondary worktree (branch name matches directory name).
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "worktree", "add", "../secondary-worktree")
	t.Cleanup(func() {
		_ = os.Chdir(mainPath)
		_, _ = util.Execute(util.ExecuteOptions{}, "git", "worktree", "remove", "--force", "../secondary-worktree")
	})
	if err := os.Chdir("../secondary-worktree"); err != nil {
		t.Fatal(err)
	}
	gitutil.ResetCacheForTesting()

	testutil.AddCommit("worktree-commit", "worktree-file")

	testParseArguments("new", "1")

	// Verify that gh pr create used the remote main branch ("main"), not the local worktree branch ("secondary-worktree").
	prCreateCall, found := findGhPrCreateCall(testExecutor.Responses)
	assert.True(found, "expected gh pr create to be called")
	baseIndex := slices.Index(prCreateCall.Args, "--base")
	assert.Greater(baseIndex, -1, "expected --base flag in gh pr create args")
	baseBranch := prCreateCall.Args[baseIndex+1]
	assert.Equal(gitutil.GetRemoteMainBranchOrDie(), baseBranch, "gh pr create --base should use remote main branch, not the worktree branch")
}

func TestSdNew_WhenNoDraft_NoReadyPromptShown(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")

	interactive.SendToProgram(0,
		// What commit do you want to create a PR from?
		interactive.NewMessageKey(tea.KeyEnter),
	)
	// No mark ready or reviewer prompts when --draft=false
	testParseArguments("new", "--draft=false")

	allCommits := templates.GetAllCommits()

	assert.True(gitutil.RemoteHasBranch(allCommits[0].Branch))
}
