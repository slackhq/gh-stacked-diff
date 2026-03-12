package interactive

import (
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tinyspeck/gh-stacked-diff/v2/util"
)

var progressIndicatorLogLineStyle = baseStyle.
	Foreground(lipgloss.Color("240"))

type progressIndicatorModel struct {
	titles       []string
	progressBars []progress.Model
	cancelled    bool
	logLines     []string
	error        any
}

var _ failableModel = progressIndicatorModel{}

func (m progressIndicatorModel) Error() any {
	return m.error
}

var _ requiredInputAware = progressIndicatorModel{}

func (m progressIndicatorModel) InputRequired() bool {
	return false
}

func (m progressIndicatorModel) Init() tea.Cmd {
	return nil
}

func (m progressIndicatorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		}
	case setProgressMsg:
		return m, m.progressBars[msg.index].SetPercent(msg.progress)
	case setLogLineMsg:
		m.logLines[msg.index] = msg.logLine
		return m, nil
	case errorMsg:
		m.error = msg.error
		return m, tea.Quit
	}
	commands := []tea.Cmd{}
	for i, progressBar := range m.progressBars {
		updatedProgressBar, cmd := progressBar.Update(msg)
		commands = append(commands, cmd)
		m.progressBars[i] = updatedProgressBar.(progress.Model)
	}
	return m, tea.Batch(commands...)
}

func (m progressIndicatorModel) View() string {
	var out strings.Builder
	for i := range m.titles {
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString(promptStyle.Render(m.titles[i]) + "\n")
		out.WriteString(m.progressBars[i].View() + "\n")
		if m.logLines[i] != "" {
			out.WriteString(progressIndicatorLogLineStyle.Render(m.logLines[i]) + "\n")
		}
	}
	return out.String()
}

type setProgressMsg struct {
	index    int
	progress float64
}

type setLogLineMsg struct {
	index   int
	logLine string
}

type ProgressIndicator struct {
	program *tea.Program
}

var _ tea.Model = progressIndicatorModel{}
var _ tea.Msg = setProgressMsg{}

// Blocks until Quit is called.
func (p *ProgressIndicator) Show(appConfig util.AppConfig) {
	finalModel := runProgram(appConfig.Io, p.program)
	if finalModel.(progressIndicatorModel).cancelled {
		appConfig.Exit(0)
	}
}

func (p *ProgressIndicator) SetProgress(index int, progress float64) {
	p.program.Send(setProgressMsg{index: index, progress: progress})
}

func (p *ProgressIndicator) SetLogLine(index int, logLine string) {
	p.program.Send(setLogLineMsg{index: index, logLine: logLine})
}

func (p *ProgressIndicator) Quit() {
	p.program.Send(tea.Quit())
}

func (p *ProgressIndicator) SendErrorOnPanic() {
	SendErrorOnPanic(p.program)
}

/*
 * Creates one progress bar for each message.
 */
func NewProgressIndicator(stdIo util.StdIo, titles []string) *ProgressIndicator {
	initialModel := progressIndicatorModel{
		titles: titles,
		progressBars: util.MapSlice(titles, func(_ string) progress.Model {
			return progress.New()
		}),
		logLines:  make([]string, len(titles)),
		cancelled: false,
	}
	return &ProgressIndicator{
		program: newProgram(initialModel, stdIo),
	}
}
