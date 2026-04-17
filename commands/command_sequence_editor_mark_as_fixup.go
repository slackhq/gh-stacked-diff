package commands

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func createMarkAsFixupCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "sequence-editor-mark-as-fixup targetCommit fixupCommit [fixupCommit...] rebaseFilename",
		Short:  "Sequence editor for git rebase used by update",
		Long:   "For use as a sequence editor during an interactive git rebase. Marks commits as fixup commits.",
		Hidden: true,
		Args:   cobra.MinimumNArgs(3),
		Annotations: map[string]string{
			checkRepoAnnotation: "true",
		},
		Run: func(cmd *cobra.Command, args []string) {
			targetCommit := args[0]
			fixupCommits := args[1 : len(args)-1]
			rebaseFilename := args[len(args)-1]
			markAsFixup(targetCommit, fixupCommits, rebaseFilename)
		},
	}
}

func markAsFixup(targetCommit string, fixupCommits []string, rebaseFilename string) {
	slog.Debug(fmt.Sprint("Got targetCommit ", targetCommit, " fixupCommits ", fixupCommits, " rebaseFilename ", rebaseFilename))

	// First pass: collect fixup lines by reading the file.
	data, err := os.ReadFile(rebaseFilename)
	if err != nil {
		panic(fmt.Sprint("Could not open ", rebaseFilename, err))
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")

	fixupLines := make([]string, 0, len(fixupCommits))
	for _, line := range lines {
		if isPickLineForCommits(line, fixupCommits) {
			fixupLines = append(fixupLines, strings.Replace(line, "pick", "fixup", 1))
		}
	}
	if len(fixupLines) != len(fixupCommits) {
		panic(fmt.Sprint("Could only find ", len(fixupLines), " of ", len(fixupCommits), " fixup commits ", fixupCommits, " in ", lines))
	}

	// Second pass: rewrite the file, moving fixup lines after the target commit.
	rewriteRebaseFile(rebaseFilename, func(line string) string {
		if isPickLineForCommits(line, fixupCommits) {
			return "" // Will be removed; these lines are re-inserted after the target.
		}
		if strings.HasPrefix(line, "pick ") && strings.Contains(line, targetCommit) {
			var buf strings.Builder
			buf.WriteString(line)
			for _, fixupLine := range fixupLines {
				buf.WriteString("\n")
				buf.WriteString(fixupLine)
			}
			return buf.String()
		}
		return line
	})
}
