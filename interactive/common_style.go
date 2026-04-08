package interactive

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

var baseStyle = lipgloss.NewStyle()

var promptStyle = baseStyle.Bold(true)

var legendCounted = map[util.LegendType]bool{}

// countLegendShown increments the shown count for the given legend type in metrics.yaml
// if it has not already been counted this process.
func countLegendShown(legend util.LegendType) {
	if !legendCounted[legend] {
		legendCounted[legend] = true
		util.IncrementLegendShownCount(legend)
	}
}
