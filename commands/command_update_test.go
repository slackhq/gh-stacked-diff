package commands

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"slices"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func TestSdUpdate_WhenAddingOlderCommit_UpdatesPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")  // oldest — to be cherry-picked
	testutil.AddCommit("second", "") // PR dest
	testParseArguments("new", "1")   // PR for "second"
	testutil.AddCommit("third", "")
	testutil.AddCommit("fourth", "")

	// NewCommits: fourth(1), third(2), second(3), first(4)
	// dest=second(3), cherrypick=first(4) — "first" is OLDER than "second"
	testParseArguments("update", "3", "4")

	allCommits := templates.GetAllCommits()
	// "first" fixup'd into "second", leaving: fourth, third, second, initial
	assert.Equal(4, len(allCommits))
	assert.Equal("fourth", allCommits[0].Subject)
	assert.Equal("third", allCommits[1].Subject)
	assert.Equal("second", allCommits[2].Subject)
	assert.Equal(testutil.InitialCommitSubject, allCommits[3].Subject)
}

func TestSdUpdate_OnRootCommit_UpdatesPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")

	commitsOnMain := templates.GetAllCommits()

	testParseArguments("update", "2", "1")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", commitsOnMain[1].Branch)

	commitsOnBranch := templates.GetAllCommits()

	assert.Equal(3, len(commitsOnBranch))
}

func TestSdUpdate_OnExistingRoot_UpdatesPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "push", "origin", gitutil.GetLocalMainBranchOrDie())

	testutil.AddCommit("second", "")

	testParseArguments("new", "1")

	testutil.AddCommit("third", "")

	testutil.AddCommit("fourth", "")

	commitsOnMain := templates.GetAllCommits()

	testParseArguments("update", "3", "1")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", commitsOnMain[2].Branch)

	allCommitsOnBranch := templates.GetAllCommits()

	assert.Equal(4, len(allCommitsOnBranch))
	assert.Equal("fourth", allCommitsOnBranch[0].Subject)
	assert.Equal("second", allCommitsOnBranch[1].Subject)
	assert.Equal("first", allCommitsOnBranch[2].Subject)
	assert.Equal(testutil.InitialCommitSubject, allCommitsOnBranch[3].Subject)

	newCommitsOnBranch := templates.GetNewCommits("HEAD", "")

	assert.Equal(2, len(newCommitsOnBranch))
	assert.Equal(newCommitsOnBranch[0].Subject, "fourth")
	assert.Equal(newCommitsOnBranch[1].Subject, "second")
}

func TestSdUpdate_UpdatesPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")

	allCommits := templates.GetAllCommits()

	testParseArguments("update", allCommits[1].Commit, "1")

	allCommits = templates.GetAllCommits()

	assert.Equal(2, len(allCommits))
	assert.Equal("first", allCommits[0].Subject)
	assert.Equal(testutil.InitialCommitSubject, allCommits[1].Subject)
}

func TestSdUpdate_WithListIndicators_UpdatesPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")
	testutil.AddCommit("third", "")

	testParseArguments("update", "3", "2", "1")

	allCommits := templates.GetAllCommits()

	assert.Equal(2, len(allCommits))
	assert.Equal("first", allCommits[0].Subject)
	assert.Equal(testutil.InitialCommitSubject, allCommits[1].Subject)

	testParseArguments("checkout", "1")
	allCommits = templates.GetAllCommits()

	assert.Equal(4, len(allCommits))
	assert.Equal("third", allCommits[0].Subject)
	assert.Equal("second", allCommits[1].Subject)
	assert.Equal("first", allCommits[2].Subject)
	assert.Equal(testutil.InitialCommitSubject, allCommits[3].Subject)
}

func TestSdUpdate_WithReviewers_AddReviewers(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")

	allCommits := templates.GetAllCommits()

	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	testParseArguments("update", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "--reviewers=mybestie", "2", "1")

	contains := slices.ContainsFunc(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		ghExpectedArgs := []string{"pr", "edit", allCommits[1].Branch, "--add-reviewer", "mybestie"}
		return next.ProgramName == "gh" && slices.Equal(next.Args, ghExpectedArgs)
	})
	assert.True(contains, util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh"
	}))
}

func TestSdUpdate_WhenCherryPickFails_RestoresBranch(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	testParseArguments("new", "1")
	testutil.CommitFileChange("second", "first", "made change")
	testutil.CommitFileChange("third", "first", "another change")

	allCommits := templates.GetAllCommits()
	defer func() {
		r := recover()
		if r != nil {
			assert.Equal(gitutil.GetLocalMainBranchOrDie(), util.GetCurrentBranchName())
			assert.Equal(allCommits, templates.GetAllCommits())
		}
	}()

	testParseArguments("update", "3", "1")

	assert.Fail("did not panic on cherry-pick")
}

func TestSdUpdate_WhenPushFails_RestoresBranches(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")
	firstBranch := templates.GetAllCommits()[0].Branch

	testParseArguments("new", "1")

	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", firstBranch)
	firstCommits := templates.GetAllCommits()
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", gitutil.GetLocalMainBranchOrDie())

	testutil.AddCommit("second", "")

	allCommits := templates.GetAllCommits()
	testExecutor.SetResponse("", errors.New("Exit code 128"), "git", "push", util.MatchAnyRemainingArgs)
	defer func() {
		r := recover()
		if r != nil {
			assert.Equal(gitutil.GetLocalMainBranchOrDie(), util.GetCurrentBranchName())
			assert.Equal(allCommits, templates.GetAllCommits())

			util.ExecuteOrDie(util.ExecuteOptions{}, "git", "switch", firstBranch)
			assert.Equal(firstCommits, templates.GetAllCommits())
		}
	}()
	testParseArguments("update", "2", "1")

	assert.Fail("did not panic on cherry-pick")
}

func TestSdUpdate_WhenRemoteBranchUpdatedConcurrently_ForceWithLeaseRejectsThePush(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	allCommits := templates.GetAllCommits()
	branchName := allCommits[0].Branch

	// Set up a concurrent clone for simulating another user's push.
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "clone", "../remote-repo", "../concurrent-repo")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-C", "../concurrent-repo", "config", "user.email", "other@example.com")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-C", "../concurrent-repo", "config", "user.name", "Other User")
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-C", "../concurrent-repo", "checkout", branchName)

	testutil.AddCommit("second", "")

	// Fake merge --ff-only to fail (triggers forcePush path).
	testExecutor.SetResponseFunc("", errors.New("Exit code 128"), func(programName string, args ...string) bool {
		return programName == "git" && slices.Contains(args, "merge") && slices.Contains(args, "--ff-only")
	})
	// Fake the rebase with "origin" upstream to succeed (can't resolve in test repos).
	testExecutor.SetResponseFunc("", nil, func(programName string, args ...string) bool {
		return programName == "git" && slices.Contains(args, "rebase") && slices.Contains(args, "origin")
	})
	// Intercept the regular push in the forcePush path: simulate a concurrent
	// push to origin as a side effect, then fail the regular push so the code
	// falls through to --force-with-lease. The concurrent push makes origin's
	// ref differ from our remote-tracking ref (set during the earlier fetch),
	// so --force-with-lease will reject the push.
	testExecutor.SetResponseFunc("", errors.New("Exit code 128"), func(programName string, args ...string) bool {
		if programName == "git" && slices.Contains(args, "push") && !slices.Contains(args, "--force-with-lease") && !slices.Contains(args, "-C") {
			// Simulate concurrent push to the same branch.
			util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-C", "../concurrent-repo", "commit", "--allow-empty", "-m", "concurrent change")
			util.ExecuteOrDie(util.ExecuteOptions{}, "git", "-C", "../concurrent-repo", "push", "-f", "origin", branchName)
			return true
		}
		return false
	})

	allCommitsBeforeUpdate := templates.GetAllCommits()
	defer func() {
		r := recover()
		if r != nil {
			assert.Equal(gitutil.GetLocalMainBranchOrDie(), util.GetCurrentBranchName())
			assert.Equal(allCommitsBeforeUpdate, templates.GetAllCommits())
		}
	}()

	testParseArguments("update", "2", "1")

	assert.Fail("Expected push to fail due to --force-with-lease rejection")
}

func TestSdUpdate_WhenCherryPickCommitsNotSpecifiedAndUserQuits_NoOp(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")

	commitsOnMain := templates.GetAllCommits()

	defer func() {
		r := recover()
		if r != nil {
			assert.Equal(commitsOnMain, templates.GetAllCommits())
		}
	}()
	// What commits do you want to add?
	interactive.SendToProgram(0, interactive.NewMessageRune('q'))
	testParseArguments("update", "2")

	assert.Fail("did not panic on quit")
}

func TestSdUpdate_WhenCherryPickCommitsNotSpecified_CherryPicsUserSelection(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")

	// What commits do you want to add?
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))
	// Mark PR as ready for review when checks pass?
	interactive.SendToProgram(1, interactive.NewMessageRune('n'))
	testParseArguments("update", "2")

	allCommits := templates.GetAllCommits()

	assert.Equal(2, len(allCommits))
	assert.Equal("first", allCommits[0].Subject)
	assert.Equal(testutil.InitialCommitSubject, allCommits[1].Subject)
}

func TestSdUpdate_WhenDestinationCommitNotSpecified_UpdatesSelectedPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")

	// What PR do you want to update?
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))
	// What commits do you want to add?
	interactive.SendToProgram(1, interactive.NewMessageKey(tea.KeyEnter))
	// Mark PR as ready for review when checks pass?
	interactive.SendToProgram(2, interactive.NewMessageRune('n'))
	testParseArguments("update")

	allCommits := templates.GetAllCommits()

	assert.Equal(2, len(allCommits))
	assert.Equal("first", allCommits[0].Subject)
	assert.Equal(testutil.InitialCommitSubject, allCommits[1].Subject)
}

func TestSdUpdate_WhenDestinationCommitNotSpecifiedAndMultiplePossibleValues_UpdatesSelectedPr(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "destination")
	testParseArguments("new", "1")
	testutil.AddCommit("second", "")
	testParseArguments("new", "1")
	testutil.AddCommit("third", "")
	testutil.AddCommit("fourth", "added1")
	testutil.AddCommit("fifth", "added2")
	testutil.AddCommit("sixth", "")

	// What PR do you want to update?
	interactive.SendToProgram(0,
		interactive.NewMessageKey(tea.KeyDown),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	// What commits do you want to add?
	interactive.SendToProgram(1,
		interactive.NewMessageKey(tea.KeyDown),
		interactive.NewMessageRune(' '),
		interactive.NewMessageKey(tea.KeyDown),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	// Mark PR as ready for review when checks pass?
	interactive.SendToProgram(2, interactive.NewMessageRune('n'))
	testParseArguments("update")

	allCommits := templates.GetAllCommits()

	assert.Equal(5, len(allCommits))
	assert.Equal("sixth", allCommits[0].Subject)
	assert.Equal("third", allCommits[1].Subject)
	assert.Equal("second", allCommits[2].Subject)
	assert.Equal("first", allCommits[3].Subject)
	assert.Equal(testutil.InitialCommitSubject, allCommits[4].Subject)
	assert.True(gitutil.RemoteHasBranch(allCommits[3].Branch))
}

func TestSdUpdate_WhenBranchAlreadyMergedAndUserDoesNotConfirm_Cancels(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")

	allCommits := templates.GetAllCommits()

	// Are you sure you want to update this PR?
	interactive.SendToProgram(0, interactive.NewMessageRune('n'))
	testExecutor.SetResponse(allCommits[1].Branch+" fakeMergeCommit",
		nil, "gh", "pr", "list", util.MatchAnyRemainingArgs)

	defer func() {
		r := recover()
		if r != nil {
			assert.Equal(allCommits, templates.GetAllCommits())
		}
	}()

	testParseArguments("update", "2", "1")

	assert.Fail("did not cancel")
}

func TestSdUpdate_WhenBranchAlreadyMergedAndUserConfirms_Updates(t *testing.T) {
	assert := assert.New(t)
	testExecutor := testutil.InitTest(t, slog.LevelError)
	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")

	allCommits := templates.GetAllCommits()
	testExecutor.SetResponse(allCommits[1].Branch+" fakeMergeCommit",
		nil, "gh", "pr", "list", util.MatchAnyRemainingArgs)

	// Are you sure you want to update this PR?
	interactive.SendToProgram(0, interactive.NewMessageRune('y'))
	testParseArguments("update", "2", "1")

	assert.Equal(2, len(templates.GetAllCommits()))
}

func TestSdUpdate_WhenNoReviewers_ConfirmReady_MarksPrReady(t *testing.T) {
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")

	testExecutor.SetResponse(
		// Each check has 3 values: status, conclusion, and state. Copied DefaultMinChecks times.
		strings.Repeat("SUCCESS\nSUCCESS\nSUCCESS\n", gitutil.DefaultMinChecks),
		nil, "gh", "pr", "view", util.MatchAnyRemainingArgs)

	// What commits do you want to add?
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))
	// Mark PR as ready for review when checks pass?
	interactive.SendToProgram(1, interactive.NewMessageRune('y'))
	// Reviewers to add when checks pass?
	interactive.SendToProgram(2, interactive.NewMessageKey(tea.KeyEnter))
	testParseArguments("update", "--min-checks", fmt.Sprint(gitutil.DefaultMinChecks), "2")

	allCommits := templates.GetAllCommits()

	// Verify gh pr ready was called
	assertGhSubcommandCalled(t, testExecutor.Responses, true, "pr", "ready", allCommits[0].Branch)

	// Verify no reviewers were added (no gh pr edit --add-reviewer)
	assertGhSubcommandCalled(t, testExecutor.Responses, false, "pr", "edit")
}

func TestSdUpdate_ConfigPromptForReview(t *testing.T) {
	for _, tt := range promptForReviewTestCases {
		t.Run(tt.name, func(t *testing.T) {
			runPromptForReviewTest(t, tt, func(_ *util.TestExecutor) []string {
				testutil.AddCommit("first", "")
				testParseArguments("new", "1")
				testutil.AddCommit("second", "")
				return []string{"update", "2"}
			})
		})
	}
}

func TestSdUpdate_WhenNoReviewers_DeclineReady_PrStaysDraft(t *testing.T) {
	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testutil.AddCommit("second", "")

	// What commits do you want to add?
	interactive.SendToProgram(0, interactive.NewMessageKey(tea.KeyEnter))
	// Mark PR as ready for review when checks pass?
	interactive.SendToProgram(1, interactive.NewMessageRune('n'))
	testParseArguments("update", "2")

	// Verify gh pr ready was NOT called
	assertGhSubcommandCalled(t, testExecutor.Responses, false, "pr", "ready")
}
