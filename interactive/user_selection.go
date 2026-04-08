package interactive

import (
	"regexp"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

var ReviewersHistory = util.NewHistoricalData("reviewers.history", 30)
var allCollaboratorsHistory = util.NewHistoricalData("all-collaborators.cache", -1)

type userSelectionModel struct {
	textInput     textinput.Model
	history       []string
	suggestions   []string
	breakingChars []rune
	historyIndex  int
	confirmed     bool
	windowWidth   int

	loadedSuggestions         bool
	loadingSuggestionsSpinner spinner.Model
	error                     any
}

var _ failableModel = userSelectionModel{}

func (m userSelectionModel) Error() any {
	return m.error
}

func (m userSelectionModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.loadingSuggestionsSpinner.Tick)
}

func (m userSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			m.confirmed = true
			return m, tea.Quit
		case tea.KeyUp:
			m.onKeyUp()
			return m, nil
		case tea.KeyDown:
			m.onKeyDown()
			return m, nil
		}
	case setSuggestionsMsg:
		m.loadedSuggestions = true
		m.suggestions = msg.suggestions
		m.setSuggestions()
		return m, nil
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		return m, nil
	case errorMsg:
		m.error = msg.error
		return m, tea.Quit
	}

	m.setSuggestions()

	commands := []tea.Cmd{}
	previousValue := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)
	commands = append(commands, cmd)
	updatedValue := m.textInput.Value()
	if previousValue != updatedValue {
		m.historyIndex = -1
	}
	m.loadingSuggestionsSpinner, cmd = m.loadingSuggestionsSpinner.Update(msg)
	commands = append(commands, cmd)
	return m, tea.Batch(commands...)
}

func (m userSelectionModel) View() string {
	if m.confirmed {
		return ""
	}
	userPrefix := "   users   "
	if m.loadedSuggestions {
		userPrefix += "  "
	} else {
		userPrefix += m.loadingSuggestionsSpinner.View() + " "
	}
	users := strings.Join(m.getMatchingSuggestions(), " ")
	if len(users)+len(userPrefix) > m.windowWidth {
		users = users[0:min(max(0, m.windowWidth-len(userPrefix)), len(users))]
	}
	users = userPrefix + users + "\n"
	result := promptStyle.Render("Reviewers to add when checks pass?") + "\n" +
		m.textInput.View() + "\n"
	if util.GetUserConfig().ShowUiLegend {
		countUiLegendShown()
		result += "\n" +
			"Controls:\n" +
			"   up/down   history\n" +
			"   tab       select auto-complete\n" +
			"   enter     confirm\n" +
			"   esc       quit\n" +
			"   comma     use comma or space to separate reviewers\n"
	}
	return result + users
}

// Returns the suggestions that match the current wip text.
func (m *userSelectionModel) getMatchingSuggestions() []string {
	csv, wipText := m.splitInput()
	matchingSuggestions := util.FilterSlice(m.suggestions, func(next string) bool {
		// more lenient than m.textInput.MatchingSuggestions
		return strings.Contains(strings.ToUpper(next), strings.ToUpper(wipText))
	})
	selectedFields := strings.FieldsFunc(csv, func(next rune) bool {
		return slices.Contains(m.breakingChars, next)
	})
	return slices.DeleteFunc(matchingSuggestions, func(next string) bool {
		return slices.Contains(selectedFields, next)
	})
}

// Sets suggestions so users can be added to an existing comma delimited string.
func (m *userSelectionModel) setSuggestions() {
	csv, _ := m.splitInput()
	if csv != "" {
		selectedFields := strings.FieldsFunc(m.textInput.Value(), func(next rune) bool {
			return slices.Contains(m.breakingChars, next)
		})
		nonSelectedSuggestions := util.FilterSlice(m.suggestions, func(next string) bool {
			return !slices.Contains(selectedFields, next)
		})
		m.textInput.SetSuggestions(util.MapSlice(nonSelectedSuggestions, func(next string) string {
			return csv + next
		}))
	} else {
		m.textInput.SetSuggestions(m.suggestions)
	}
}

// Returns the current text input split between the CSV portion and the wip text.
func (m *userSelectionModel) splitInput() (string, string) {
	valueRunes := []rune(m.textInput.Value())
	for i := len(valueRunes) - 1; i >= 0; i-- {
		if slices.Contains(m.breakingChars, valueRunes[i]) {
			return string(valueRunes[0 : i+1]), string(valueRunes[i+1:])
		}
	}
	return "", m.textInput.Value()
}

func (m *userSelectionModel) onKeyUp() {
	appendToHistory := ""
	if m.historyIndex == -1 && m.textInput.Value() != "" && len(m.history) > 0 && m.history[len(m.history)-1] != m.textInput.Value() {
		appendToHistory = m.textInput.Value()
	}
	if m.historyIndex == -1 && len(m.history) > 0 {
		m.historyIndex = len(m.history) - 1
		m.textInput.SetValue(m.history[m.historyIndex])
		m.textInput.SetCursor(len(m.textInput.Value()))
	} else if m.historyIndex > 0 {
		m.historyIndex--
		m.textInput.SetValue(m.history[m.historyIndex])
		m.textInput.SetCursor(len(m.textInput.Value()))
	}
	if appendToHistory != "" {
		m.history = append(m.history, appendToHistory)
	}
}

func (m *userSelectionModel) onKeyDown() {
	if m.historyIndex != -1 {
		if m.historyIndex < len(m.history)-1 {
			m.historyIndex++
			m.textInput.SetValue(m.history[m.historyIndex])
			m.textInput.SetCursor(len(m.textInput.Value()))
		} else {
			m.historyIndex = -1
			m.textInput.SetValue("")
			m.textInput.SetCursor(len(m.textInput.Value()))
		}
	}
}

type setSuggestionsMsg struct {
	suggestions []string
}

var _ tea.Model = userSelectionModel{}
var _ tea.Msg = setSuggestionsMsg{}

func UserSelection() string {
	appConfig := util.GetAppConfig()
	input := textinput.New()
	input.Focus()
	input.Width = 100
	input.Placeholder = "None (mark ready only)"
	input.ShowSuggestions = true
	history := ReviewersHistory.ReadHistory()
	suggestions := allCollaboratorsHistory.ReadHistory()
	input.SetSuggestions(suggestions)
	initialModel := userSelectionModel{
		history:       history,
		historyIndex:  -1,
		textInput:     input,
		confirmed:     false,
		suggestions:   suggestions,
		breakingChars: []rune{',', ' '},

		loadedSuggestions:         false,
		loadingSuggestionsSpinner: spinner.New(),
	}
	program := newProgram(initialModel, appConfig.Io)
	go updateSuggestions(program, suggestions)
	finalModel := runProgram(appConfig.Io, program)
	finalSelectionModel := finalModel.(userSelectionModel)
	if !finalSelectionModel.confirmed {
		appConfig.Exit(0)
	}
	selected := finalSelectionModel.textInput.Value()
	return normalizeReviewers(selected)
}

func normalizeReviewers(selected string) string {
	selected = strings.ReplaceAll(selected, " ", "#")
	selected = strings.ReplaceAll(selected, ",", "#")
	expression := regexp.MustCompile("#+")
	selected = expression.ReplaceAllString(selected, ",")
	return selected
}

// Updates suggestions with results from API collaborators call.
func updateSuggestions(program *tea.Program, originalSuggestions []string) {
	defer SendErrorOnPanic(program)
	appConfig := util.GetAppConfig()
	allCollaborators := getAllCollaborators()
	if appConfig.DemoMode {
		program.Send(setSuggestionsMsg{suggestions: originalSuggestions})
	} else {
		// Do not set if in demo mode, instead write the cache file manually.
		program.Send(setSuggestionsMsg{suggestions: allCollaborators})
		allCollaboratorsHistory.SetHistory(allCollaborators)
	}
}
