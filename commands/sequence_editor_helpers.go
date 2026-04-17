package commands

import (
	"fmt"
	"os"
	"strings"
)

// rewriteRebaseFile reads a git rebase todo file, applies a transform function
// to each line, and writes the result back. If transform returns an empty string,
// the line is omitted from the output.
func rewriteRebaseFile(filename string, transform func(line string) string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(fmt.Sprint("Could not open ", filename, err))
	}

	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	var newText strings.Builder
	for _, line := range lines {
		result := transform(line)
		if result != "" {
			newText.WriteString(result)
			newText.WriteString("\n")
		}
	}

	err = os.WriteFile(filename, []byte(newText.String()), 0644)
	if err != nil {
		panic(err)
	}
}

// isPickLineForCommits returns true if the line is a "pick" line for any of the given commits.
func isPickLineForCommits(line string, commits []string) bool {
	if !strings.HasPrefix(line, "pick ") {
		return false
	}
	for _, commit := range commits {
		if strings.Contains(line, commit) {
			return true
		}
	}
	return false
}
