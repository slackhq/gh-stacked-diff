/*
For use as a sequence editor for an interactive git rebase.
Drop any commits specified in the input parameters, keep the others.

usage: sequence_editor_drop_already_merged dropCommit1 [dropCommit2...] rebaseFilename
*/
package commands

import (
	"fmt"
	"log/slog"
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

	rewriteRebaseFile(rebaseFilename, func(line string) string {
		if isPickLineForCommits(line, dropCommits) {
			return strings.Replace(line, "pick", "drop", 1)
		}
		return line
	})
}
