package gitutil

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// ghStackState is the subset of the file written by `gh stack` (<git-common-dir>/gh-stack, shared by all worktrees) that is needed
// to determine stack membership.
type ghStackState struct {
	Stacks []struct {
		Branches []struct {
			Branch string `json:"branch"`
		} `json:"branches"`
	} `json:"stacks"`
}

// GetGhStackBranches returns the branches, in stack order, of the `gh stack` stack that contains
// branchName (including branchName itself). Returns an empty list if branchName is not part of a
// stack, or if the stack file is missing or unreadable.
func GetGhStackBranches(branchName string) []string {
	gitDir, err := util.Execute(util.ExecuteOptions{}, "git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		slog.Debug(fmt.Sprint("Could not determine common git dir: ", err))
		return []string{}
	}
	return readGhStackBranches(filepath.Join(strings.TrimSpace(gitDir), "gh-stack"), branchName)
}

func readGhStackBranches(stackFilePath string, branchName string) []string {
	contents, err := os.ReadFile(stackFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn(fmt.Sprint("Could not read ", stackFilePath, ": ", err))
		}
		return []string{}
	}
	var state ghStackState
	if err := json.Unmarshal(contents, &state); err != nil {
		slog.Warn(fmt.Sprint("Could not parse ", stackFilePath, ": ", err))
		return []string{}
	}
	for _, stack := range state.Stacks {
		for _, branch := range stack.Branches {
			if branch.Branch != branchName {
				continue
			}
			names := make([]string, 0, len(stack.Branches))
			for _, b := range stack.Branches {
				names = append(names, b.Branch)
			}
			return names
		}
	}
	return []string{}
}

// GhStackLabel describes a branch's membership in a `gh stack` stack.
type GhStackLabel struct {
	// InStack is true if the branch is part of a stack.
	InStack bool
	// Label is "A", "B", ..., "Z", "AA", ... identifying the stack. Empty if the branch is not in a stack, or
	// if there is only one stack (so there is nothing to distinguish).
	Label string
}

// GetGhStackLabels returns the stack label for each of the given branches. Branches are in `sd log` order
// (newest first), and labels are assigned from the bottom (last element), so the oldest stacked branch gets "A".
// Branches in the same stack share a label.
func GetGhStackLabels(branches []string) []GhStackLabel {
	labels := make([]GhStackLabel, len(branches))
	labelsByStack := make(map[string]string)
	for i := len(branches) - 1; i >= 0; i-- {
		stackBranches := GetGhStackBranches(branches[i])
		if len(stackBranches) == 0 {
			continue
		}
		stackKey := strings.Join(stackBranches, "\n")
		label, ok := labelsByStack[stackKey]
		if !ok {
			label = stackLabel(len(labelsByStack))
			labelsByStack[stackKey] = label
		}
		labels[i] = GhStackLabel{InStack: true, Label: label}
	}
	if len(labelsByStack) == 1 {
		for i := range labels {
			labels[i].Label = ""
		}
	}
	return labels
}

// stackLabel returns the spreadsheet-style column name for a zero-based index: A..Z, AA, AB, ...
func stackLabel(index int) string {
	label := ""
	for n := index + 1; n > 0; n = (n - 1) / 26 {
		label = string(rune('A'+(n-1)%26)) + label
	}
	return label
}
