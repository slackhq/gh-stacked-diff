package interactive

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChoiceModel_ViewNumbersOptions(t *testing.T) {
	model := choiceModel{
		options: []string{"first", "second", "third"},
		cursor:  0,
		prompt:  "Choose one:",
	}

	view := ansi.Strip(model.View())

	assert.Equal(t, "Choose one:\n ▸ 1. first\n   2. second\n   3. third\n", view)
}

func TestChoiceModel_UpdateSelectsOptionByNumber(t *testing.T) {
	model := choiceModel{
		options:  []string{"first", "second", "third"},
		selected: -1,
	}

	updated, cmd := model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'2'}}))
	result := updated.(choiceModel)

	require.NotNil(t, cmd)
	assert.Equal(t, 1, result.selected)
	assert.True(t, result.completed)
}

func TestChoiceModel_UpdateIgnoresNumberWithoutMatchingOption(t *testing.T) {
	model := choiceModel{
		options:  []string{"first"},
		selected: -1,
	}

	updated, cmd := model.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'2'}}))
	result := updated.(choiceModel)

	assert.Nil(t, cmd)
	assert.Equal(t, -1, result.selected)
	assert.False(t, result.completed)
}

func TestChoiceModel_ViewHasOneOptionPerLine(t *testing.T) {
	model := choiceModel{
		options: []string{"first", "second"},
		prompt:  "Choose one:",
	}

	assert.Len(t, strings.Split(strings.TrimSuffix(ansi.Strip(model.View()), "\n"), "\n"), 3)
}
