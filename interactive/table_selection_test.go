package interactive

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestCommitSelector_MultiLineSummaryNotTruncated(t *testing.T) {
	util.SetUserConfig(util.UserConfig{})
	columns := []string{"Index", "Commit", "Summary"}
	rows := [][]string{
		{"1", "abc123", "Main commit subject"},
	}
	selector := NewCommitSelector("Select:", columns, rows, false, func(row int) bool { return true }, "")

	multiLineSummary := "Main commit subject\n- Fix review issues\n- Use domain types"
	updated, _ := selector.Update(updateCommitSelectorSummaryMsg{Index: 0, Summary: multiLineSummary})
	rendered := ansi.Strip(updated.(CommitSelector).View())

	for _, line := range strings.Split(multiLineSummary, "\n") {
		assert.Contains(t, rendered, line, "rendered table should contain line: %q", line)
	}
}
