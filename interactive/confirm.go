package interactive

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

type confirmModel struct {
	prompt     string
	defaultYes bool
	confirmed  bool
	cancelled  bool
}

var _ tea.Model = confirmModel{}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "n", "N":
			return m, tea.Quit
		case "enter":
			m.confirmed = m.defaultYes
			return m, tea.Quit
		case "y", "Y":
			m.confirmed = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.defaultYes {
		return promptStyle.Render(m.prompt) + " (" + promptStyle.Render("Y") + "/n): "
	}
	return promptStyle.Render(m.prompt) + " (y/" + promptStyle.Render("N") + "): "
}

func Confirm(prompt string, defaultYes bool) bool {
	appConfig := util.GetAppConfig()
	initialModel := confirmModel{prompt: prompt, defaultYes: defaultYes}
	finalModel := runProgram(appConfig.Io, newProgram(initialModel, appConfig.Io))
	if finalModel.(confirmModel).cancelled {
		appConfig.Exit(0)
	}
	return finalModel.(confirmModel).confirmed
}

func ConfirmOrDie(prompt string, defaultYes bool) {
	if !Confirm(prompt, defaultYes) {
		util.GetAppConfig().Exit(0)
	}
}
