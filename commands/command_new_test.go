package commands

import (
	"bytes"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"testing"

	"github.com/stretchr/testify/assert"

	"errors"

	"github.com/tinyspeck/gh-stacked-diff/v2/interactive"
	"github.com/tinyspeck/gh-stacked-diff/v2/templates"
	"github.com/tinyspeck/gh-stacked-diff/v2/testutil"
	"github.com/tinyspeck/gh-stacked-diff/v2/util"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSdNew_OnRepoWithPreviousCommit_CreatesPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())

	testutil.AddCommit("second", "")
	allCommits := templates.GetNewCommits("HEAD")

	testParseArguments("new", "1")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", allCommits[0].Branch)
	commitsOnNewBranch := templates.GetNewCommits("HEAD")
	assert.Equal(1, len(commitsOnNewBranch))
	assert.Equal(allCommits[0].Subject, commitsOnNewBranch[0].Subject)
}

func TestSdNew_WithMiddleCommit_CreatesPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", util.GetMainBranchOrDie())

	testutil.AddCommit("second", "")

	testutil.AddCommit("third", "")
	allCommits := templates.GetNewCommits("HEAD")

	testParseArguments("new", "1")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", allCommits[0].Branch)
	commitsOnNewBranch := templates.GetNewCommits("HEAD")
	assert.Equal(1, len(commitsOnNewBranch))
	assert.Equal(allCommits[0].Subject, commitsOnNewBranch[0].Subject)
}

func TestSdNew_CreatesPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	out := testParseArguments("log")

	assert.Contains(out, "✅")
}

func TestSdNew_WithReviewers_AddReviewers(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", util.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	testParseArguments("new", "--min-checks", fmt.Sprint(util.DefaultMinChecks), "--reviewers=mybestie", "1")

	allCommits := templates.GetAllCommits()

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "edit", allCommits[0].Branch, "--add-reviewer", "mybestie"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
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

	assert.Equal(true, util.RemoteHasBranch(allCommits[1].Branch))
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

	assert.True(util.RemoteHasBranch(allCommits[1].Branch))
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

	assert.True(util.RemoteHasBranch(allCommits[2].Branch))
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
	assert.True(util.RemoteHasBranch(allCommits[2].Branch))
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
	assert.True(util.RemoteHasBranch(allCommits[1].Branch))
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

	assert.True(util.RemoteHasBranch(allCommits[0].Branch))
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
			assert.Contains(out.String(), "│ 1    │")
			assert.Contains(out.String(), "│10 ✅ │")
		}
	}()
	testParseArgumentsWithOut(out, "--log-level=error", "new")
	assert.Fail("did not panic on cancel")
}

func TestSdNew_WhenNoReviewersAndDraft_ConfirmReady_MarksPrReady(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", util.DefaultMinChecks),
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
	testParseArguments("new", "--min-checks", fmt.Sprint(util.DefaultMinChecks))

	allCommits := templates.GetAllCommits()

	// Verify gh pr ready was called
	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh" && len(next.Args) >= 3 &&
			next.Args[0] == "pr" && next.Args[1] == "ready" && next.Args[2] == allCommits[0].Branch
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))

	// Verify no reviewers were added (no gh pr edit --add-reviewer)
	containsReviewer := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh" && len(next.Args) >= 2 &&
			next.Args[0] == "pr" && next.Args[1] == "edit"
	})
	assert.False(containsReviewer)
}

func TestSdNew_WhenNoReviewersAndDraft_EnterDefaultsToReady_MarksPrReady(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", util.DefaultMinChecks),
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
	testParseArguments("new", "--min-checks", fmt.Sprint(util.DefaultMinChecks))

	allCommits := templates.GetAllCommits()

	// Verify gh pr ready was called
	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh" && len(next.Args) >= 3 &&
			next.Args[0] == "pr" && next.Args[1] == "ready" && next.Args[2] == allCommits[0].Branch
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}

func TestSdNew_WhenNoReviewersAndDraft_DeclineReady_PrStaysDraft(t *testing.T) {
	assert := assert.New(t)

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
	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh" && len(next.Args) >= 2 &&
			next.Args[0] == "pr" && next.Args[1] == "ready"
	})
	assert.False(contains)
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
			assert.Equal(util.GetMainBranchOrDie(), util.GetCurrentBranchName())
			assert.Equal(allCommitsBeforeNew, templates.GetAllCommits())
		}
	}()

	testParseArguments("new", "1")

	assert.Fail("Expected push to fail due to --force-with-lease rejection")
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

	assert.True(util.RemoteHasBranch(allCommits[0].Branch))
}
