package commands

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func createMarkAsFixupCommand() Command {
	flagSet := flag.NewFlagSet("sequence-editor-mark-as-fixup", flag.ContinueOnError)
	return Command{
		FlagSet:     flagSet,
		Summary:     "Sequence editor for git rebase used by update",
		Description: "For use as a sequence editor during an interactive git rebase. Marks commits as fixup commits.",
		Usage:       "sd " + flagSet.Name() + " targetCommit fixupCommit1 [fixupCommit2...] rebaseFilename",
		Hidden:      true,
		OnSelected: func(appConfig util.AppConfig, command Command) {
			if flagSet.NArg() < 3 {
				commandError(appConfig, flagSet, "not enough arguments", command.Usage)
			}

			targetCommit := flagSet.Arg(0)
			fixupCommits := flagSet.Args()[1 : len(flagSet.Args())-1]
			rebaseFilename := flagSet.Arg(len(flagSet.Args()) - 1)

			markAsFixup(targetCommit, fixupCommits, rebaseFilename)
		}}
}

func markAsFixup(targetCommit string, fixupCommits []string, rebaseFilename string) {
	data, err := os.ReadFile(rebaseFilename)

	slog.Debug(fmt.Sprint("Got targetCommit ", targetCommit, " fixupCommits ", fixupCommits, " rebaseFilename ", rebaseFilename))
	if err != nil {
		panic(fmt.Sprint("Could not open ", rebaseFilename, err))
	}

	originalText := string(data)
	var newText strings.Builder

	fixupLines := make([]string, len(fixupCommits))
	i := 0

	lines := strings.Split(strings.TrimSuffix(originalText, "\n"), "\n")
	for _, line := range lines {
		if isFixupLine(line, fixupCommits) {
			fixupLines[i] = strings.Replace(line, "pick", "fixup", 1)
			i++
		}
	}
	if i != len(fixupCommits) {
		panic(fmt.Sprint("Could only find ", i, " of ", len(fixupCommits), " fixup commits ", fixupCommits, " in ", lines))
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "pick ") && strings.Contains(line, targetCommit) {
			newText.WriteString(line)
			newText.WriteString("\n")
			for _, fixupLine := range fixupLines {
				newText.WriteString(fixupLine)
				newText.WriteString("\n")
			}
		} else if !isFixupLine(line, fixupCommits) {
			newText.WriteString(line)
			newText.WriteString("\n")
		}
	}

	err = os.WriteFile(rebaseFilename, []byte(newText.String()), 0)
	if err != nil {
		panic(err)
	}
}

func isFixupLine(line string, fixupCommits []string) bool {
	if !strings.HasPrefix(line, "pick ") {
		return false
	}
	for _, commit := range fixupCommits {
		if strings.Contains(line, commit) {
			return true
		}
	}
	return false
}
