package interactive

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	cachedStatus := util.PullRequestStatus{State: util.PullRequestStateOpen, CanMerge: true}
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
	updated, _ := m.Update(updateAllRowsMsg{rows: newRows})
	model := updated.(logStatusModel)
	assert.Equal(t, "test commit updated", model.rows[0].log.Subject)
	assert.NotNil(t, model.rows[0].status)
	assert.True(t, model.rows[0].status.CanMerge)
	assert.Len(t, model.rows[0].branchCommits, 1)
}

func TestLogStatusModel_UpdateStatusRow_SetsStatus(t *testing.T) {
	m := newTestModel(true)
	status := util.PullRequestStatus{State: util.PullRequestStateMerged}
	updated, _ := m.Update(updateLogStatusRowMsg{index: 0, status: status})
	model := updated.(logStatusModel)
	assert.NotNil(t, model.rows[0].status)
	assert.Equal(t, util.PullRequestStateMerged, model.rows[0].status.State)
}

func TestLogStatusModel_UpdateStatusRow_IgnoresOutOfBounds(t *testing.T) {
	m := newTestModel(true)
	status := util.PullRequestStatus{State: util.PullRequestStateMerged}
	updated, _ := m.Update(updateLogStatusRowMsg{index: 99, status: status})
	model := updated.(logStatusModel)
	assert.Nil(t, model.rows[0].status)
}

func TestFormatBranchCommits_EmptySlice(t *testing.T) {
	assert.Equal(t, "", FormatBranchCommits(nil, "  "))
	assert.Equal(t, "", FormatBranchCommits([]templates.GitLog{}, "  "))
}

func TestFormatBranchCommits_SingleCommit(t *testing.T) {
	commits := []templates.GitLog{{Subject: "only one"}}
	assert.Equal(t, "", FormatBranchCommits(commits, "  "))
}
