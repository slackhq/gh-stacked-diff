package interactive

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

const maxNumberedChoiceOptions = 9

type choiceModel struct {
	options   []string
	cursor    int
	selected  int
	completed bool
	prompt    string
	header    string
}

var _ tea.Model = choiceModel{}

func (m choiceModel) Init() tea.Cmd { return nil }

func (m choiceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "Q", "ctrl+c":
			m.selected = -1
			return m, tea.Quit
		case "enter":
			m.selected = m.cursor
			m.completed = true
			return m, tea.Quit
		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.options) - 1
			}
		case "down", "j":
			m.cursor++
			if m.cursor >= len(m.options) {
				m.cursor = 0
			}
		default:
			if len(m.options) <= maxNumberedChoiceOptions &&
				len(msg.String()) == 1 && msg.String()[0] >= '1' && msg.String()[0] <= '9' {
				selected := int(msg.String()[0] - '1')
				if selected < len(m.options) {
					m.selected = selected
					m.completed = true
					return m, tea.Quit
				}
			}
		}
	}
	return m, nil
}

func (m choiceModel) View() string {
	var b strings.Builder
	if m.header != "" {
		b.WriteString(m.header + "\n")
	}
	if m.completed {
		return b.String()
	}
	b.WriteString(promptStyle.Render(m.prompt) + "\n")
	numbered := len(m.options) <= maxNumberedChoiceOptions
	for i, option := range m.options {
		label := option
		if numbered {
			label = fmt.Sprintf("%d. %s", i+1, option)
		}
		if i == m.cursor {
			b.WriteString(highlightEnabledStyle.Render(" ▸ " + label))
		} else {
			b.WriteString("   " + label)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// GetChoiceSelection displays an interactive single-select list for arbitrary string options.
// The header, if not empty, is shown above the prompt and stays visible after a selection is made.
//
// Returns the 0-based index of the selected option, and false if cancelled.
func GetChoiceSelection(header string, prompt string, options []string) (int, bool) {
	appConfig := util.GetAppConfig()
	model := choiceModel{
		header:   header,
		options:  options,
		cursor:   0,
		selected: -1,
		prompt:   prompt,
	}
	program := newProgram(model, appConfig.Io)
	finalModel := runProgram(appConfig.Io, program).(choiceModel)
	return finalModel.selected, finalModel.selected != -1
}
