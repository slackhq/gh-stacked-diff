package interactive

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

type dashboardRow struct {
	index  string
	pr     bool
	status *util.PullRequestStatus
	log    templates.GitLog
}

type dashboardModel struct {
	spinner spinner.Model
	table   table.Model
	rows    []dashboardRow
	error   any
}

var _ failableModel = dashboardModel{}

func (m dashboardModel) Error() any {
	return m.error
}

func (m dashboardModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case updateDashboardRowMsg:
		m.rows[msg.index] = msg.row
		return m, nil
	case errorMsg:
		m.error = msg.error
		return m, tea.Quit
	}
	var tableCmd tea.Cmd
	m.table, tableCmd = m.table.Update(msg)
	var spinnerCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)
	return m, tea.Batch(tableCmd, spinnerCmd)
}

func (m dashboardModel) View() string {
	m.table.SetRows(m.getTableRows())
	if m.table.Cursor() == -1 {
		m.table.SetCursor(0)
	}
	return m.table.View() + "\n"
}

func (m dashboardModel) getTableRows() []table.Row {
	tableRows := make([]table.Row, len(m.rows))
	for i, row := range m.rows {
		pr := ""
		checksPassed := ""
		approved := ""
		if row.pr {
			pr = "pr"
			if row.status != nil {
				if row.status.Checks.IsSuccess() {
					checksPassed = "passesd"
				} else if row.status.Checks.IsFailing() {
					checksPassed = "failed"
				} else {
					checksPassed = fmt.Sprint(row.status.Checks.PercentageComplete())
				}
				approved = strings.Join(row.status.Approvers, "\n")
			} else {
				checksPassed = m.spinner.View()
				approved = m.spinner.View()
			}
		}
		tableRows[i] = table.Row{row.index, pr, checksPassed, approved, row.log.Commit, row.log.Subject + "\nnext 2"}
	}
	return tableRows
}

type updateDashboardRowMsg struct {
	index int
	row   dashboardRow
}

var _ tea.Model = dashboardModel{}
var _ tea.Msg = updateDashboardRowMsg{}

func ShowDashboard(appConfig util.AppConfig, minChecks int) {

	columns := []string{"Index", "PR", "Checks", "Approved", "Commit", "Summary"}
	newCommits := templates.GetNewCommits("HEAD")
	gitBranchArgs := make([]string, 0, len(newCommits)+2)
	gitBranchArgs = append(gitBranchArgs, "branch", "-l")
	for _, log := range newCommits {
		gitBranchArgs = append(gitBranchArgs, log.Branch)
	}
	prBranches := strings.Fields(util.ExecuteOrDie(util.ExecuteOptions{}, "git", gitBranchArgs...))

	rows := make([]dashboardRow, len(newCommits))
	for i, log := range newCommits {
		hasLocalBranch := slices.Contains(prBranches, log.Branch)
		indexString := fmt.Sprint(i + 1)
		paddingLen := len(fmt.Sprint(len(newCommits))) - len(indexString)
		indexString = strings.Repeat(" ", paddingLen) + indexString
		rows[i] = dashboardRow{
			index: indexString, pr: hasLocalBranch, log: log, status: nil,
		}
	}

	tableColumns := util.MapSlice(columns, func(columnName string) table.Column {
		return table.Column{Title: columnName}
	})
	t := table.New(
		table.WithColumns(tableColumns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithWrapCursor(true),
	)
	initialModel := dashboardModel{
		spinner: spinner.New(),
		table:   t,
		rows:    rows,
	}
	initialModel.spinner.Spinner = spinner.Dot
	program := newProgram(initialModel, appConfig.Io)
	go updateDashboardData(appConfig, program, rows, minChecks)
	runProgram(appConfig.Io, program)
	// finalModel := runProgram(appConfig.Io, program)
	// finalDashboardModel := finalModel.(dashboardModel)
	// println("finalDashboardModel", fmt.Sprint(finalDashboardModel))
}

func updateDashboardData(appConfig util.AppConfig, program *tea.Program, rows []dashboardRow, minChecks int) {
	defer SendErrorOnPanic(program)
	for i, row := range rows {
		if row.pr {
			status := util.GetPullRequestStatus(appConfig, row.log.Branch, minChecks)
			row.status = &status
			program.Send(updateDashboardRowMsg{index: i, row: row})
		}
	}
}
