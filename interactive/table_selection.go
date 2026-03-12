package interactive

import (
	"slices"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tinyspeck/gh-stacked-diff/v2/util"
)

var highlightEnabledStyle = baseStyle.
	Foreground(lipgloss.Color("229")).
	Background(lipgloss.Color("57"))

var highlightDisabledStyle = baseStyle.
	Foreground(lipgloss.Color("240")).
	Background(lipgloss.Color("244"))

var enabledRowStyle = baseStyle

var disabledRowStyle = baseStyle.
	Foreground(lipgloss.Color("240"))

var selectedRowStyle = baseStyle.Bold(true)

var selectedHighlightRowStyle = highlightEnabledStyle.Bold(true)

type CommitSelector struct {
	table        table.Model
	SelectedRows []int
	multiselect  bool
	rowEnabled   func(row int) bool
	completed    bool
	prompt       string
}

var _ tea.Model = CommitSelector{}

type UpdateCommitSelectorSummaryMsg struct {
	Index   int
	Summary string
}

var _ tea.Msg = UpdateCommitSelectorSummaryMsg{}

func (m CommitSelector) Init() tea.Cmd { return nil }

func (m CommitSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		direction := 0
		switch msg.String() {
		case "esc", "q", "Q", "ctrl+c":
			m.SelectedRows = []int{}
			return m, tea.Quit
		case " ":
			if !m.multiselect || !m.rowEnabled(m.table.Cursor()) {
				break
			}
			existingIndex := slices.Index(m.SelectedRows, m.table.Cursor())
			if existingIndex != -1 {
				m.SelectedRows = slices.Delete(m.SelectedRows, existingIndex, existingIndex+1)
			} else {
				m.SelectedRows = append(m.SelectedRows, m.table.Cursor())
			}
			return m, nil
		case "enter":
			if !m.rowEnabled(m.table.Cursor()) {
				break
			}
			if !slices.Contains(m.SelectedRows, m.table.Cursor()) {
				m.SelectedRows = append(m.SelectedRows, m.table.Cursor())
			}
			m.completed = true
			return m, tea.Quit
		case "up", "k":
			direction = -1
		case "down", "j":
			direction = 1
		}
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		if direction != 0 && !m.rowEnabled(m.table.Cursor()) {
			m.skipDisabledRows(direction)
		}
		return m, cmd
	case tea.WindowSizeMsg:
		m.table.SetHeight(min(max(msg.Height-10, 5), 20))
		return m, nil
	case UpdateCommitSelectorSummaryMsg:
		rows := m.table.Rows()
		rows[msg.Index][2] = msg.Summary
		m.table.SetRows(rows)
		return m, nil
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// skipDisabledRows moves the cursor by step (-1 or 1) until it lands on
// an enabled row or wraps back to the starting position.
func (m *CommitSelector) skipDisabledRows(step int) {
	rowCount := len(m.table.Rows())
	if rowCount == 0 {
		return
	}
	start := m.table.Cursor()
	next := start
	for {
		next = (next + step + rowCount) % rowCount
		if m.rowEnabled(next) || next == start {
			break
		}
	}
	m.table.SetCursor(next)
}

// This needs to be recreated everytime the model changes so that the model reference is updated.
func (m CommitSelector) createStyleFunc() func(tableModel table.Model, row int, col int) lipgloss.Style {
	return func(tableModel table.Model, row int, col int) lipgloss.Style {
		switch {
		case row < 0 || row >= len(tableModel.Rows()):
			// < 0 is the header row
			// >= len can happen on resize
			return baseStyle
		case row == tableModel.Cursor():
			if m.rowEnabled(row) {
				if slices.Contains(m.SelectedRows, row) {
					return selectedHighlightRowStyle
				} else {
					return highlightEnabledStyle
				}
			} else {
				return highlightDisabledStyle
			}
		case slices.Contains(m.SelectedRows, row):
			return selectedRowStyle
		default:
			if m.rowEnabled(row) {
				return enabledRowStyle
			} else {
				return disabledRowStyle
			}
		}
	}
}

func (m CommitSelector) View() string {
	if m.completed {
		return ""
	}
	m.table.SetStyleFunc(m.createStyleFunc())
	return promptStyle.Render(m.prompt) + "\n" + m.table.View() + "\n"
}

func NewCommitSelector(
	prompt string,
	columns []string,
	rows [][]string,
	multiselect bool,
	rowEnabled func(row int) bool,
) CommitSelector {
	tableColumns := util.MapSlice(columns, func(columnName string) table.Column {
		return table.Column{Title: columnName}
	})

	tableRows := make([]table.Row, len(rows))
	firstEnabledRow := -1
	for i, rowData := range rows {
		tableRows[i] = rowData
		if firstEnabledRow == -1 && rowEnabled(i) {
			firstEnabledRow = i
		}
	}
	t := table.New(
		table.WithColumns(tableColumns),
		table.WithRows(tableRows),
		table.WithFocused(true),
		table.WithWrapCursor(true),
	)
	if firstEnabledRow != -1 {
		t.SetCursor(firstEnabledRow)
	}
	return CommitSelector{
		SelectedRows: []int{},
		table:        t,
		multiselect:  multiselect,
		rowEnabled:   rowEnabled,
		prompt:       prompt,
	}
}
