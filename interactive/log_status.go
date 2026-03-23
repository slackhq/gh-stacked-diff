package interactive

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

var (
	grayColor   = color.New().AddRGB(128, 128, 128)
	purpleColor = color.New().AddRGB(128, 0, 128)
	hidingColor = color.New(color.Italic).AddRGB(88, 88, 88)
)

type logStatusRow struct {
	log           templates.GitLog
	hasPR         bool
	status        *util.PullRequestStatus
	branchCommits []templates.GitLog
	numberPrefix  string
	padding       string
}

type logStatusModel struct {
	spinner spinner.Model
	rows    []logStatusRow
	polling bool
	loading bool
	error   any
}

var _ failableModel = logStatusModel{}
var _ tea.Model = logStatusModel{}
var _ requiredInputAware = logStatusModel{}

func (m logStatusModel) Error() any {
	return m.error
}

func (m logStatusModel) InputRequired() bool {
	return false
}

func (m logStatusModel) Init() tea.Cmd {
	return m.spinner.Tick
}

type updateLogStatusRowMsg struct {
	index  int
	status util.PullRequestStatus
}

type updateLogStatusBranchCommitsMsg struct {
	index         int
	branchCommits []templates.GitLog
}

type updateAllRowsMsg struct {
	rows []logStatusRow
}

type pollFetchStartMsg struct{}
type allStatusesLoadedMsg struct{}

func (m logStatusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case pollFetchStartMsg:
		m.loading = true
		return m, nil
	case updateAllRowsMsg:
		// Carry over cached status and branch commits from old rows
		// so that previously loaded data continues to display while refreshing.
		oldByBranch := make(map[string]*logStatusRow, len(m.rows))
		for i := range m.rows {
			oldByBranch[m.rows[i].log.Branch] = &m.rows[i]
		}
		for i := range msg.rows {
			if old, ok := oldByBranch[msg.rows[i].log.Branch]; ok {
				msg.rows[i].status = old.status
				msg.rows[i].branchCommits = old.branchCommits
			}
		}
		m.rows = msg.rows
		return m, nil
	case updateLogStatusRowMsg:
		if msg.index < len(m.rows) {
			m.rows[msg.index].status = &msg.status
		}
		return m, nil
	case updateLogStatusBranchCommitsMsg:
		if msg.index < len(m.rows) {
			m.rows[msg.index].branchCommits = msg.branchCommits
		}
		return m, nil
	case allStatusesLoadedMsg:
		m.loading = false
		if !m.polling {
			return m, tea.Quit
		}
		return m, nil
	case errorMsg:
		m.error = msg.error
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m logStatusModel) View() string {
	var out strings.Builder
	for _, row := range m.rows {
		if row.hasPR {
			out.WriteString(row.numberPrefix + color.GreenString("✅ "))
		} else {
			out.WriteString(row.numberPrefix + "   ")
		}
		out.WriteString(color.YellowString(row.log.Commit) + " " + row.log.Subject + "\n")
		if row.hasPR {
			statusLine := row.padding + m.formatStatus(row.status)
			if row.status != nil {
				reviewInfo := formatReviewSummary(row.status)
				if reviewInfo != "" {
					statusLine += " " + reviewInfo
				}
			}
			out.WriteString(statusLine + "\n")
		}
		if row.hasPR && len(row.branchCommits) > 1 {
			out.WriteString(FormatBranchCommits(row.branchCommits, row.padding))
		}
	}
	if m.polling && m.loading && !m.hasInlineSpinner() {
		out.WriteString(m.spinner.View() + "\n")
	}
	return out.String()
}

func (m logStatusModel) hasInlineSpinner() bool {
	for _, row := range m.rows {
		if row.hasPR && row.status == nil {
			return true
		}
	}
	return false
}

func (m logStatusModel) formatStatus(status *util.PullRequestStatus) string {
	if status == nil {
		return m.spinner.View()
	}
	var parts []string
	switch status.State {
	case util.PullRequestStateOpen:
		parts = append(parts, color.CyanString("[open]"))
	case util.PullRequestStateMerged:
		parts = append(parts, purpleColor.Sprint("[merged]"))
	case util.PullRequestStateClosed:
		parts = append(parts, color.RedString("[closed]"))
	}
	if status.IsDraft {
		parts = append(parts, grayColor.Sprint("[draft]"))
	}
	checks := status.Checks
	total := checks.Total()
	if checks.IsSuccess() {
		parts = append(parts, color.GreenString(fmt.Sprintf("[checks: passed (%d/%d)]", checks.Passing, total)))
	} else if checks.IsFailing() {
		parts = append(parts, color.RedString(fmt.Sprintf("[checks: failed (%d/%d failed)]", checks.Failing, total)))
	} else if total > 0 {
		parts = append(parts, color.YellowString(fmt.Sprintf("[checks: pending (%d/%d passed)]", checks.Passing, total)))
	} else {
		parts = append(parts, "[checks: none]")
	}
	approvedCount := 0
	for _, review := range status.LatestReviews {
		if review.State == "APPROVED" && !review.HasComments() {
			approvedCount++
		}
	}
	if approvedCount > 0 {
		parts = append(parts, color.GreenString(fmt.Sprintf("[approved: %d/%d]", approvedCount, status.TotalReviewers)))
	} else {
		parts = append(parts, fmt.Sprintf("[approved: %d/%d]", approvedCount, status.TotalReviewers))
	}
	if status.CanMerge {
		parts = append(parts, color.GreenString("[can merge]"))
	}
	return strings.Join(parts, " ")
}

func getReviewStatusKey(review util.LatestReview) string {
	switch review.State {
	case "APPROVED":
		if review.HasComments() {
			return "approvedWithComments"
		}
		return "approved"
	case "CHANGES_REQUESTED":
		return "changesRequested"
	case "COMMENTED":
		return "commented"
	default:
		return ""
	}
}

func formatReviewSummary(status *util.PullRequestStatus) string {
	// Group logins by status key.
	groupLogins := make(map[string][]string)
	for _, review := range status.LatestReviews {
		key := getReviewStatusKey(review)
		if key == "" {
			continue
		}
		groupLogins[key] = append(groupLogins[key], review.Login)
	}
	keyOrder := []string{"changesRequested", "commented", "approvedWithComments", "approved"}
	var parts []string
	for _, key := range keyOrder {
		logins, ok := groupLogins[key]
		if !ok {
			continue
		}
		slices.SortFunc(logins, func(a, b string) int {
			return strings.Compare(strings.ToLower(a), strings.ToLower(b))
		})
		names := strings.Join(logins, ", ")
		switch key {
		case "approved":
			parts = append(parts, color.GreenString(names+" approved"))
		case "approvedWithComments":
			// Using red even though it is approved so that the user does not miss that they should check the review comments.
			parts = append(parts, color.RedString(names+" approved with comments"))
		case "changesRequested":
			parts = append(parts, color.RedString(names+" requested changes"))
		case "commented":
			parts = append(parts, color.RedString(names+" commented"))
		}
	}
	return strings.Join(parts, " | ")
}

// LogDataFunc returns the current logs and checked branches.
// It is called to refresh the log data on each poll iteration.
type LogDataFunc func() ([]templates.GitLog, []string)

func ShowLogStatus(logs []templates.GitLog, checkedBranches []string, pollInterval time.Duration, refreshFunc LogDataFunc) {
	appConfig := util.GetAppConfig()
	rows := buildRows(logs, checkedBranches)
	polling := pollInterval > 0
	s := spinner.New()
	s.Spinner = spinner.Dot
	initialModel := logStatusModel{
		spinner: s,
		rows:    rows,
		polling: polling,
		loading: true,
	}
	program := newProgram(initialModel, appConfig.Io)
	go fetchAllStatuses(program, rows, polling, pollInterval, refreshFunc)
	runProgram(appConfig.Io, program)
}

func buildRows(logs []templates.GitLog, checkedBranches []string) []logStatusRow {
	rows := make([]logStatusRow, len(logs))
	for i, log := range logs {
		prefix := GetLogNumberPrefix(i, len(logs))
		rows[i] = logStatusRow{
			log:          log,
			hasPR:        slices.Contains(checkedBranches, log.Branch),
			numberPrefix: prefix,
			padding:      strings.Repeat(" ", len(prefix)),
		}
	}
	return rows
}

func fetchAllStatuses(program *tea.Program, rows []logStatusRow, polling bool, pollInterval time.Duration, refreshFunc LogDataFunc) {
	defer SendErrorOnPanic(program)
	for {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 2)
		for i, row := range rows {
			if row.hasPR {
				wg.Add(1)
				go func() {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					branchCommits := templates.GetNewCommits(row.log.Branch)
					program.Send(updateLogStatusBranchCommitsMsg{index: i, branchCommits: branchCommits})
					// Use 1 as minChecks is ignored in this flow.
					status := util.GetPullRequestStatus(row.log.Branch, 1)
					program.Send(updateLogStatusRowMsg{index: i, status: status})
				}()
			}
		}
		wg.Wait()
		program.Send(allStatusesLoadedMsg{})
		if !polling {
			return
		}
		time.Sleep(pollInterval)
		program.Send(pollFetchStartMsg{})
		// Refresh the full log data (new commits, new PRs, etc.)
		logs, checkedBranches := refreshFunc()
		rows = buildRows(logs, checkedBranches)
		program.Send(updateAllRowsMsg{rows: rows})
	}
}

// FormatBranchCommits formats the additional commits on a branch for display.
// It copies the slice, reverses to chronological order, skips the first (which
// matches the main log entry), and truncates with a hiding message if needed.
func FormatBranchCommits(branchCommits []templates.GitLog, padding string) string {
	if len(branchCommits) <= 1 {
		return ""
	}
	commits := make([]templates.GitLog, len(branchCommits))
	copy(commits, branchCommits)
	slices.Reverse(commits)
	commits = commits[1:]
	var out strings.Builder
	if len(commits) > 3 {
		hidingMessage := hidingColor.Sprint("   - [hiding ", (len(commits) - 2), " previous...]")
		out.WriteString(padding + hidingMessage + "\n")
		commits = commits[len(commits)-2:]
	}
	for _, bc := range commits {
		out.WriteString(padding + "   - " + bc.Subject + "\n")
	}
	return out.String()
}

func GetLogNumberPrefix(i int, numLogs int) string {
	maxIndex := fmt.Sprint(numLogs)
	currentIndex := fmt.Sprint(i + 1)
	padding := strings.Repeat(" ", len(maxIndex)-len(currentIndex))
	return padding + currentIndex + ". "
}
