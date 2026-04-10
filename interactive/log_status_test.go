package interactive

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func newTestModel(polling bool) logStatusModel {
	return logStatusModel{
		rows: []logStatusRow{
			{
				log:          templates.GitLog{Commit: "abc123", Subject: "test commit", Branch: "test-branch"},
				hasPR:        true,
				numberPrefix: "1. ",
				padding:      "   ",
			},
		},
		polling: polling,
		loading: true,
	}
}

func TestLogStatusModel_AllStatusesLoaded_QuitsWhenNotPolling(t *testing.T) {
	m := newTestModel(false)
	_, cmd := m.Update(allStatusesLoadedMsg{})
	assert.NotNil(t, cmd)
	// tea.Quit returns a special quit message.
	msg := cmd()
	assert.Equal(t, tea.QuitMsg{}, msg)
}

func TestLogStatusModel_AllStatusesLoaded_ContinuesWhenPolling(t *testing.T) {
	m := newTestModel(true)
	updated, cmd := m.Update(allStatusesLoadedMsg{})
	assert.Nil(t, cmd)
	assert.False(t, updated.(logStatusModel).loading)
}

func TestLogStatusModel_PollFetchStart_SetsLoading(t *testing.T) {
	m := newTestModel(true)
	m.loading = false
	updated, _ := m.Update(pollFetchStartMsg{})
	assert.True(t, updated.(logStatusModel).loading)
}

func TestLogStatusModel_UpdateAllRows_CarriesOverCachedStatus(t *testing.T) {
	m := newTestModel(true)
	cachedStatus := gitutil.PullRequestStatus{State: gitutil.PullRequestStateOpen, CanMerge: true}
	m.rows[0].status = &cachedStatus
	m.rows[0].branchCommits = []templates.GitLog{{Subject: "extra"}}

	newRows := []logStatusRow{
		{
			log:          templates.GitLog{Commit: "abc123", Subject: "test commit updated", Branch: "test-branch"},
			hasPR:        true,
			numberPrefix: "1. ",
			padding:      "   ",
		},
	}
	updated, _ := m.Update(updateAllRowsMsg{rows: newRows, generation: 1})
	model := updated.(logStatusModel)
	assert.Equal(t, 1, model.generation)
	assert.Equal(t, "test commit updated", model.rows[0].log.Subject)
	assert.NotNil(t, model.rows[0].status)
	assert.True(t, model.rows[0].status.CanMerge)
	assert.Len(t, model.rows[0].branchCommits, 1)
}

func TestLogStatusModel_UpdateStatusRow_SetsStatus(t *testing.T) {
	m := newTestModel(true)
	status := gitutil.PullRequestStatus{State: gitutil.PullRequestStateMerged}
	updated, _ := m.Update(updateLogStatusRowMsg{index: 0, status: status})
	model := updated.(logStatusModel)
	assert.NotNil(t, model.rows[0].status)
	assert.Equal(t, gitutil.PullRequestStateMerged, model.rows[0].status.State)
}

func TestLogStatusModel_UpdateStatusRow_IgnoresOutOfBounds(t *testing.T) {
	m := newTestModel(true)
	status := gitutil.PullRequestStatus{State: gitutil.PullRequestStateMerged}
	updated, _ := m.Update(updateLogStatusRowMsg{index: 99, status: status})
	model := updated.(logStatusModel)
	assert.Nil(t, model.rows[0].status)
}

func TestLogStatusModel_UpdateStatusRow_IgnoresStaleGeneration(t *testing.T) {
	m := newTestModel(true)
	m.generation = 2
	status := gitutil.PullRequestStatus{State: gitutil.PullRequestStateMerged}
	updated, _ := m.Update(updateLogStatusRowMsg{index: 0, status: status, generation: 1})
	model := updated.(logStatusModel)
	assert.Nil(t, model.rows[0].status)
}

func TestLogStatusModel_UpdateBranchCommits_IgnoresStaleGeneration(t *testing.T) {
	m := newTestModel(true)
	m.generation = 2
	updated, _ := m.Update(updateLogStatusBranchCommitsMsg{index: 0, branchCommits: []templates.GitLog{{Subject: "stale"}}, generation: 1})
	model := updated.(logStatusModel)
	assert.Nil(t, model.rows[0].branchCommits)
}

func newTestRows(count int) []logStatusRow {
	rows := make([]logStatusRow, count)
	for i := range count {
		rows[i] = logStatusRow{
			log:          templates.GitLog{Commit: fmt.Sprintf("abc%d", i), Subject: fmt.Sprintf("commit %d", i), Branch: fmt.Sprintf("branch-%d", i)},
			numberPrefix: fmt.Sprintf("%d. ", i+1),
			padding:      "   ",
		}
	}
	return rows
}

func TestLogStatusModel_View_TruncatesWhenExceedingTerminalHeight(t *testing.T) {
	util.SetUserConfig(util.UserConfig{})
	// Each non-PR row renders as 1 line. With terminalHeight=5 and 2 reserved
	// lines (hiding message + trailing newline), we can fit 3 rows.
	m := logStatusModel{
		rows:           newTestRows(5),
		terminalHeight: 5,
	}
	view := m.View()
	assert.Contains(t, view, "commit 0")
	assert.Contains(t, view, "commit 1")
	assert.Contains(t, view, "commit 2")
	assert.NotContains(t, view, "commit 3")
	assert.NotContains(t, view, "commit 4")
	assert.Contains(t, view, "[hiding 2 more...]")
}

func TestLogStatusModel_View_NoTruncationWhenAllRowsFit(t *testing.T) {
	util.SetUserConfig(util.UserConfig{})
	m := logStatusModel{
		rows:           newTestRows(3),
		terminalHeight: 10,
	}
	view := m.View()
	assert.Contains(t, view, "commit 0")
	assert.Contains(t, view, "commit 1")
	assert.Contains(t, view, "commit 2")
	assert.NotContains(t, view, "hiding")
}

func TestLogStatusModel_View_NoTruncationWhenTerminalHeightUnknown(t *testing.T) {
	util.SetUserConfig(util.UserConfig{})
	m := logStatusModel{
		rows:           newTestRows(5),
		terminalHeight: 0,
	}
	view := m.View()
	assert.Contains(t, view, "commit 0")
	assert.Contains(t, view, "commit 4")
	assert.NotContains(t, view, "hiding")
}

func TestLogStatusModel_View_TruncatesWithPRStatusRows(t *testing.T) {
	util.SetUserConfig(util.UserConfig{})
	// PR rows render as 2 lines (commit + status), so they consume more space.
	rows := newTestRows(4)
	rows[0].hasPR = true
	rows[1].hasPR = true
	m := logStatusModel{
		rows:           rows,
		terminalHeight: 6,
	}
	view := m.View()
	// Row 0 takes 2 lines, row 1 takes 2 lines = 4 lines, maxLines = 4.
	assert.Contains(t, view, "commit 0")
	assert.Contains(t, view, "commit 1")
	assert.NotContains(t, view, "commit 2")
	assert.Contains(t, view, "[hiding 2 more...]")
}

func TestFormatBranchCommits_EmptySlice(t *testing.T) {
	assert.Equal(t, "", FormatBranchCommits(nil, "  "))
	assert.Equal(t, "", FormatBranchCommits([]templates.GitLog{}, "  "))
}

func TestFormatBranchCommits_SingleCommit(t *testing.T) {
	commits := []templates.GitLog{{Subject: "only one"}}
	assert.Equal(t, "", FormatBranchCommits(commits, "  "))
}

func TestFormatBranchCommits_MultipleCommits(t *testing.T) {
	// Commits are in reverse-chronological order (newest first), matching GetNewCommits output.
	// First commit after reverse is skipped (it matches the main log entry).
	commits := []templates.GitLog{
		{Subject: "third"},
		{Subject: "second"},
		{Subject: "first (main entry)"},
	}
	result := FormatBranchCommits(commits, "  ")
	assert.Contains(t, result, "   - second\n")
	assert.Contains(t, result, "   - third\n")
	assert.NotContains(t, result, "first (main entry)")
}

func TestFormatBranchCommits_TruncatesWhenManyCommits(t *testing.T) {
	commits := []templates.GitLog{
		{Subject: "fifth"},
		{Subject: "fourth"},
		{Subject: "third"},
		{Subject: "second"},
		{Subject: "first (main entry)"},
	}
	result := FormatBranchCommits(commits, "  ")
	assert.Contains(t, result, "[hiding 2 previous...]")
	assert.NotContains(t, result, "second")
	assert.NotContains(t, result, "third")
	assert.Contains(t, result, "   - fourth\n")
	assert.Contains(t, result, "   - fifth\n")
	assert.NotContains(t, result, "first (main entry)")
}
