package commands

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

const prBodyBegin = "This is a first PR\nWith lots of info\n"
const prBodyEnd = "\n#### Ticket: JIRA-100\n\nFeature Flag: FF1"

func TestSdAddDescription_WhenHasPr_UpdatesPrBody(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	testParseArguments("new", "1")

	testExecutor.SetResponse("101"+gitutil.GhDelim+"first"+gitutil.GhDelim+"open"+gitutil.GhDelim+prBodyBegin+prBodyEnd, nil, "gh", "pr", "list", util.MatchAnyRemainingArgs)

	generatedDescription := "Ai\ngenerated\ndescription\n"
	aiOut := "Based on the PR changes, here's my recommended description:\n```markdown\n" + generatedDescription + "\n```\n"
	testExecutor.SetResponseFunc(aiOut, nil, func(programName string, args ...string) bool {
		if programName == "a" && args[0] == "i" {
			return true
		}
		return false
	})

	// AI Command
	interactive.SendToProgram(0,
		interactive.NewMessageRune('a'),
		interactive.NewMessageRune(' '),
		interactive.NewMessageRune('i'),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	// Select commit
	interactive.SendToProgram(1, interactive.NewMessageKey(tea.KeyEnter))

	testParseArguments("add-description")

	addition := GENERATED_CLAUDE_SUMMARY_BEGIN + generatedDescription + GENERATED_CLAUDE_SUMMARY_END
	expectedBody := prBodyBegin + addition + prBodyEnd
	editCommand := util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh" && next.Args[0] == "pr" && next.Args[1] == "edit"
	})[0]
	assert.Equal(expectedBody, editCommand.Args[len(editCommand.Args)-1])
}

func TestSdAddDescription_WhenHasPrHasExistingComment_ReplacesComment(t *testing.T) {
	assert := assert.New(t)

	testExecutor := testutil.InitTest(t, slog.LevelError)

	testutil.AddCommit("first", "")

	generatedDescription := "Ai\ngenerated\ndescription\n"
	addition := GENERATED_CLAUDE_SUMMARY_BEGIN + generatedDescription + GENERATED_CLAUDE_SUMMARY_END
	originalBody := prBodyBegin + addition + prBodyEnd
	testParseArguments("new", "1")

	testExecutor.SetResponse("101"+gitutil.GhDelim+"first"+gitutil.GhDelim+"open"+gitutil.GhDelim+originalBody, nil, "gh", "pr", "list", util.MatchAnyRemainingArgs)

	updatedDescription := "this is an\nupdated description\n"
	aiOut := "Based on the PR changes, here's my recommended description:\n```markdown\n" + updatedDescription + "\n```\n"

	testExecutor.SetResponseFunc(aiOut, nil, func(programName string, args ...string) bool {
		if programName == "a" && args[0] == "i" {
			return true
		}
		return false
	})

	// AI Command
	interactive.SendToProgram(0,
		interactive.NewMessageRune('a'),
		interactive.NewMessageRune(' '),
		interactive.NewMessageRune('i'),
		interactive.NewMessageKey(tea.KeyEnter),
	)
	// Select commit
	interactive.SendToProgram(1, interactive.NewMessageKey(tea.KeyEnter))

	testParseArguments("add-description")

	expectedBody := prBodyBegin + GENERATED_CLAUDE_SUMMARY_BEGIN + updatedDescription + GENERATED_CLAUDE_SUMMARY_END + prBodyEnd
	editCommand := util.FilterSlice(testExecutor.Responses, func(next util.ExecutedResponse) bool {
		return next.ProgramName == "gh" && next.Args[0] == "pr" && next.Args[1] == "edit"
	})[0]
	assert.Equal(expectedBody, editCommand.Args[len(editCommand.Args)-1])
}
