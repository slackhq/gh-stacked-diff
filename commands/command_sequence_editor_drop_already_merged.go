/*
For use as a sequence editor for an interactive git rebase.
Drop any commits specified in the input parameters, keep the others.

usage: sequence_editor_drop_already_merged dropCommit1 [dropCommit2...] rebaseFilename
*/
package commands

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func createDropAlreadyMergedCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "sequence-editor-drop-already-merged [dropCommit...] rebaseFilename",
		Short:  "Sequence editor for git rebase used by rebase-main",
		Long:   "Drops any commits passed as arguments.",
		Hidden: true,
		Args:   cobra.MinimumNArgs(2),
		Annotations: map[string]string{
			checkRepoAnnotation: "true",
		},
		Run: func(cmd *cobra.Command, args []string) {
			dropCommits := args[0 : len(args)-1]
			rebaseFilename := args[len(args)-1]
			dropAlreadyMerged(dropCommits, rebaseFilename)
		},
	}
}

func dropAlreadyMerged(dropCommits []string, rebaseFilename string) {
	slog.Debug(fmt.Sprint("Got dropCommits ", dropCommits, " rebaseFilename ", rebaseFilename))

	data, err := os.ReadFile(rebaseFilename)
	if err != nil {
		panic(fmt.Sprint("Could not open ", rebaseFilename, err))
	}

	originalText := string(data)
	var newText strings.Builder
	// yeah the only way I can do that is via a bash script?
	i := 0
	lines := strings.Split(strings.TrimSuffix(originalText, "\n"), "\n")
	for _, line := range lines {
		if isDropLine(line, dropCommits) {
			dropLine := strings.Replace(line, "pick", "drop", 1)
			newText.WriteString(dropLine)
			newText.WriteString("\n")
			i++
		} else {
			newText.WriteString(line)
			newText.WriteString("\n")
		}
	}

	err = os.WriteFile(rebaseFilename, []byte(newText.String()), 0)
	if err != nil {
		panic(err)
	}
}

func isDropLine(line string, dropCommits []string) bool {
	if !strings.HasPrefix(line, "pick ") {
		return false
	}
	for _, commit := range dropCommits {
		if strings.Contains(line, commit) {
			return true
		}
	}
	return false
}
