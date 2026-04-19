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
	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

const maxParallelAPICalls = 2

var (
	grayColor   = color.New().AddRGB(128, 128, 128)
	purpleColor = color.New().AddRGB(128, 0, 128)
	hidingColor = color.New(color.Italic).AddRGB(88, 88, 88)
)

// WorktreeLogSection represents commits from another worktree directory.
type WorktreeLogSection struct {
	DirName         string
	Logs            []templates.GitLog
	CheckedBranches []string
}

type logStatusRow struct {
	sectionHeader string // if non-empty, rendered as a bold header before this row
	log           templates.GitLog
	hasPR         bool
	status        *gitutil.PullRequestStatus
	branchCommits []templates.GitLog
	numberPrefix  string
	padding       string
}

type logStatusModel struct {
	spinner        spinner.Model
	rows           []logStatusRow
	polling        bool
	loading        bool
	error          any
	generation     int
	terminalHeight int
}

var _ failableModel = logStatusModel{}
var _ tea.Model = logStatusModel{}
var _ requiredInputAware = logStatusModel{}

func (m logStatusModel) Error() any {
	return m.error
}

func (m logStatusModel) InputRequired() bool {
	return m.polling
}

func (m logStatusModel) Init() tea.Cmd {
	return m.spinner.Tick
}

type updateLogStatusRowMsg struct {
	index      int
	status     gitutil.PullRequestStatus
	generation int
}

type updateLogStatusBranchCommitsMsg struct {
	index         int
	branchCommits []templates.GitLog
	generation    int
}

type updateAllRowsMsg struct {
	rows       []logStatusRow
	generation int
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
		m.generation = msg.generation
		return m, nil
	case updateLogStatusRowMsg:
		if msg.generation == m.generation && msg.index < len(m.rows) {
			m.rows[msg.index].status = &msg.status
		}
		return m, nil
	case updateLogStatusBranchCommitsMsg:
		if msg.generation == m.generation && msg.index < len(m.rows) {
			m.rows[msg.index].branchCommits = msg.branchCommits
		}
		return m, nil
	case allStatusesLoadedMsg:
		m.loading = false
		if !m.polling {
			return m, tea.Quit
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.terminalHeight = msg.Height
		return m, nil
	case errorMsg:
		m.error = msg.error
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m logStatusModel) renderRow(row logStatusRow) string {
	var out strings.Builder
	if row.sectionHeader != "" {
		out.WriteString("\n" + color.New(color.Bold).Sprint(row.sectionHeader) + "\n")
	}
	if row.log.HasDuplicate {
		out.WriteString(row.numberPrefix + color.YellowString("● "))
	} else if row.hasPR {
		out.WriteString(row.numberPrefix + color.GreenString("✓ "))
	} else {
		out.WriteString(row.numberPrefix + "  ")
	}
	out.WriteString(coloredCommit(row) + " " + row.log.Subject + "\n")
	if row.hasPR {
		statusLine := row.padding + "  " + m.formatStatus(row.status)
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
	return out.String()
}

func (m logStatusModel) View() string {
	var out strings.Builder
	hasDuplicates := false
	// Reserve lines for footer elements (hiding message, legend, spinner).
	reservedLines := 2 // hiding message + trailing newline
	if m.polling && m.loading && !m.hasInlineSpinner() {
		reservedLines++
	}
	showLegend := util.GetUserConfig().ShowDuplicateSubjectLegend
	maxLines := m.terminalHeight - reservedLines
	totalLines := 0
	hiddenRows := 0
	for i, row := range m.rows {
		if row.log.HasDuplicate {
			hasDuplicates = true
		}
		rendered := m.renderRow(row)
		lineCount := strings.Count(rendered, "\n")
		legendLines := 0
		if hasDuplicates && showLegend {
			legendLines = 1 // DuplicateSubjectLegend is a single line
		}
		if m.terminalHeight > 0 && totalLines+lineCount+legendLines > maxLines {
			hiddenRows = len(m.rows) - i
			break
		}
		totalLines += lineCount
		out.WriteString(rendered)
	}
	if hasDuplicates && util.GetUserConfig().ShowDuplicateSubjectLegend {
		countLegendShown(util.LegendDuplicateSubject)
		out.WriteString(color.YellowString(templates.DuplicateSubjectLegend) + "\n")
	}
	if hiddenRows > 0 {
		out.WriteString(hidingColor.Sprint(fmt.Sprintf("[hiding %d more...]", hiddenRows)) + "\n")
	}
	if m.polling && m.loading && !m.hasInlineSpinner() {
		// Note: do not use eol here.
		out.WriteString(m.spinner.View())
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

func coloredCommit(row logStatusRow) string {
	if row.status != nil {
		if row.status.IsInMergeQueue {
			return color.YellowString(row.log.Commit)
		}
		switch row.status.State {
		case gitutil.PullRequestStateMerged:
			return purpleColor.Sprint(row.log.Commit)
		case gitutil.PullRequestStateClosed:
			return color.RedString(row.log.Commit)
		case gitutil.PullRequestStateOpen:
			if row.status.IsDraft {
				return grayColor.Sprint(row.log.Commit)
			}
			return color.CyanString(row.log.Commit)
		}
	}
	return color.YellowString(row.log.Commit)
}

func (m logStatusModel) formatStatus(status *gitutil.PullRequestStatus) string {
	if status == nil {
		return m.spinner.View()
	}
	var parts []string
	if status.IsInMergeQueue && status.CanMerge {
		parts = append(parts, color.YellowString("[merging]"))
	} else {
		switch status.State {
		case gitutil.PullRequestStateOpen:
			if status.IsDraft {
				parts = append(parts, grayColor.Sprint("[draft]"))
			} else {
				parts = append(parts, color.CyanString("[open]"))
			}
		case gitutil.PullRequestStateMerged:
			parts = append(parts, purpleColor.Sprint("[merged]"))
		case gitutil.PullRequestStateClosed:
			parts = append(parts, color.RedString("[closed]"))
		}
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
		if review.State == gitutil.ReviewStateApproved && !review.HasComments() {
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

func getReviewStatusKey(review gitutil.LatestReview) string {
	switch review.State {
	case gitutil.ReviewStateApproved:
		if review.HasComments() {
			return "approvedWithComments"
		}
		return "approved"
	case gitutil.ReviewStateChangesRequested:
		return "changesRequested"
	case gitutil.ReviewStateCommented:
		return "commented"
	default:
		return ""
	}
}

func formatReviewSummary(status *gitutil.PullRequestStatus) string {
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

// LogDataFunc returns the current logs, checked branches, and worktree sections.
// It is called to refresh the log data on each poll iteration.
type LogDataFunc func() ([]templates.GitLog, []string, []WorktreeLogSection)

func ShowLogStatus(logs []templates.GitLog, checkedBranches []string, pollInterval time.Duration, refreshFunc LogDataFunc, worktreeSections []WorktreeLogSection) {
	appConfig := util.GetAppConfig()
	rows := buildRows(logs, checkedBranches, worktreeSections)
	polling := pollInterval > 0
	hasAnyPR := false
	for _, row := range rows {
		if row.hasPR {
			hasAnyPR = true
			break
		}
	}
	if !polling && !hasAnyPR {
		// No PRs to fetch status for; print static output without bubbletea.
		m := logStatusModel{rows: rows}
		util.Fprint(appConfig.Io.Out, m.View())
		return
	}
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

func buildRows(logs []templates.GitLog, checkedBranches []string, worktreeSections []WorktreeLogSection) []logStatusRow {
	rows := appendLogRows(make([]logStatusRow, 0, len(logs)), logs, checkedBranches, "")
	for _, section := range worktreeSections {
		rows = appendLogRows(rows, section.Logs, section.CheckedBranches, section.DirName)
	}
	return rows
}

func appendLogRows(rows []logStatusRow, logs []templates.GitLog, checkedBranches []string, sectionHeader string) []logStatusRow {
	for i, log := range logs {
		prefix := GetLogNumberPrefix(i, len(logs))
		row := logStatusRow{
			log:          log,
			hasPR:        slices.Contains(checkedBranches, log.Branch),
			numberPrefix: prefix,
			padding:      strings.Repeat(" ", len(prefix)),
		}
		if i == 0 && sectionHeader != "" {
			row.sectionHeader = sectionHeader
		}
		rows = append(rows, row)
	}
	return rows
}

func fetchAllStatuses(program *tea.Program, rows []logStatusRow, polling bool, pollInterval time.Duration, refreshFunc LogDataFunc) {
	defer SendErrorOnPanic(program)
	generation := 0
	for {
		gen := generation
		var wg sync.WaitGroup
		sem := make(chan struct{}, maxParallelAPICalls)
		// Each goroutine captures i and row by value (Go 1.22+ loop semantics).
		// rows is reassigned on the next poll cycle (line below), but that
		// happens only after wg.Wait() returns, so there is no race.
		for i, row := range rows {
			if row.hasPR {
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer SendErrorOnPanic(program)
					// Branch commits are a local git operation — fetch without semaphore.
					branchCommits := templates.GetNewCommits(row.log.Branch, "")
					program.Send(updateLogStatusBranchCommitsMsg{index: i, branchCommits: branchCommits, generation: gen})
					// API calls are rate-limited by semaphore.
					sem <- struct{}{}
					defer func() { <-sem }()
					// Use 1 for minChecks as this flow does not need to have it calculated.
					status := gitutil.GetPullRequestStatus(row.log.Branch, 1)
					program.Send(updateLogStatusRowMsg{index: i, status: status, generation: gen})
				}()
			}
		}
		wg.Wait()
		program.Send(allStatusesLoadedMsg{})
		if !polling {
			return
		}
		time.Sleep(pollInterval)
		generation++
		program.Send(pollFetchStartMsg{})
		// Refresh the full log data (new commits, new PRs, etc.)
		logs, checkedBranches, worktreeSections := refreshFunc()
		rows = buildRows(logs, checkedBranches, worktreeSections)
		program.Send(updateAllRowsMsg{rows: rows, generation: generation})
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
		hidingMessage := hidingColor.Sprint("  - [hiding ", (len(commits) - 2), " previous...]")
		out.WriteString(padding + hidingMessage + "\n")
		commits = commits[len(commits)-2:]
	}
	for _, bc := range commits {
		out.WriteString(padding + "  - " + bc.Subject + "\n")
	}
	return out.String()
}

func GetLogNumberPrefix(i int, numLogs int) string {
	maxIndex := fmt.Sprint(numLogs)
	currentIndex := fmt.Sprint(i + 1)
	padding := strings.Repeat(" ", len(maxIndex)-len(currentIndex))
	return padding + currentIndex + ". "
}
