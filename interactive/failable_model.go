package interactive

import (
	tea "github.com/charmbracelet/bubbletea"
)

type failableModel interface {
	Error() any
}

type errorMsg struct {
	error any
}

// Returns a function that send error to the program instead of printing and calling os.Exit,
// as otherwise the console can be left in state where new lines are only displayed as line
// feeds (on Mac).
func SendErrorOnPanic(program *tea.Program) {
	r := recover()
	if r != nil {
		program.Send(errorMsg{error: r})
	}
}
