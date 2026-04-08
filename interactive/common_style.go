package interactive

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

var baseStyle = lipgloss.NewStyle()

var promptStyle = baseStyle.Bold(true)

var uiLegendCounted bool

// countUiLegendShown increments the shown count in metrics.yaml if it has not already been shown this process.
func countUiLegendShown() {
	if !uiLegendCounted {
		uiLegendCounted = true
		util.IncrementUiLegendShownCount()
	}
}
