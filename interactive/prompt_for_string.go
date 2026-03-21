package interactive

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

type stringModel struct {
	textInput textinput.Model
	prompt    string
	confirmed bool
}

var _ tea.Model = stringModel{}

func (m stringModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink)
}

/*
 */
func (m stringModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			m.confirmed = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m stringModel) View() string {
	if m.confirmed {
		return ""
	}
	return promptStyle.Render(m.prompt) + " " + m.textInput.View() + "\n" +
		"\n" +
		"Controls:\n" +
		"   tab       select auto-complete\n" +
		"   enter     confirm\n" +
		"   esc       quit\n"
}

func PromptForString(prompt string, suggestions []string) string {
	appConfig := util.GetAppConfig()
	input := textinput.New()
	input.Focus()
	input.Width = 30
	input.ShowSuggestions = true
	input.SetSuggestions(suggestions)
	initialModel := stringModel{
		textInput: input,
		prompt:    prompt,
	}
	finalModel := runProgram(appConfig.Io, newProgram(initialModel, appConfig.Io))
	if finalModel.(stringModel).confirmed {
		return finalModel.(stringModel).textInput.Value()
	} else {
		return ""
	}
}

func PromptForStringOrDie(prompt string, suggestions []string) string {
	result := PromptForString(prompt, suggestions)
	if result == "" {
		util.GetAppConfig().Exit(0)
	}
	return result
}
