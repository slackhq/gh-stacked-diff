package gitutil

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var codeOwnersLocations = []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"}

// Returns the "org/team" names of all teams listed in the CODEOWNERS file of the current
// directory, sorted and without duplicates. Returns empty if there is no CODEOWNERS file.
func GetCodeOwnersTeams() []string {
	for _, location := range codeOwnersLocations {
		content, err := os.ReadFile(filepath.FromSlash(location))
		if err == nil {
			return parseCodeOwnersTeams(string(content))
		}
	}
	slog.Debug("No CODEOWNERS file found")
	return []string{}
}

func parseCodeOwnersTeams(content string) []string {
	teams := []string{}
	for _, line := range strings.Split(content, "\n") {
		if comment := strings.Index(line, "#"); comment >= 0 {
			line = line[:comment]
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// First field is the file pattern.
		for _, owner := range fields[1:] {
			if strings.HasPrefix(owner, "@") && strings.Contains(owner, "/") {
				teams = append(teams, strings.TrimPrefix(owner, "@"))
			}
		}
	}
	slices.Sort(teams)
	return slices.Compact(teams)
}
