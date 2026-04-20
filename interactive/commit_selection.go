package interactive

import (
	"context"
	"fmt"
	"slices"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"

	"errors"
	"strings"
)

type CommitType int

const (
	CommitTypePr CommitType = iota
	CommitTypeNoPr
	CommitTypeBoth
)

type CommitSelectionOptions struct {
	CommitType       CommitType
	MultiSelect      bool
	Prompt           string
	GitDir           string
	DisabledBranches map[string]bool
}

// Returns an empty array if user cancelled.
func GetCommitSelection(options CommitSelectionOptions) ([]templates.GitLog, error) {
	appConfig := util.GetAppConfig()
	columns := []string{"Index", "Commit", "Summary"}
	newCommits := templates.GetNewCommits("HEAD", options.GitDir)
	branchNames := make([]string, len(newCommits))
	for i, log := range newCommits {
		branchNames[i] = log.Branch
	}
	prBranches := gitutil.CheckLocalBranches(options.GitDir, branchNames)

	rows := make([][]string, 0, len(newCommits))

	rowEnabled := func(row int) bool {
		if options.DisabledBranches[newCommits[row].Branch] {
			return false
		}
		if options.CommitType == CommitTypeBoth {
			return true
		}
		hasLocalBranch := slices.Contains(prBranches, newCommits[row].Branch)
		return (options.CommitType == CommitTypePr && hasLocalBranch) || (options.CommitType == CommitTypeNoPr && !hasLocalBranch)
	}

	hasEnabledRow := false
	hasDuplicates := false
	for i, commit := range newCommits {
		hasLocalBranch := slices.Contains(prBranches, commit.Branch)
		indexString := fmt.Sprint(i + 1)
		paddingLen := len(fmt.Sprint(len(newCommits))) - len(indexString)
		indexString = strings.Repeat(" ", paddingLen) + indexString
		if commit.HasDuplicate {
			hasDuplicates = true
			indexString += " " + color.YellowString("●")
		} else if hasLocalBranch {
			indexString += " " + color.GreenString("✓")
		}
		row := []string{indexString, commit.Commit, commit.Subject}
		if rowEnabled(i) {
			hasEnabledRow = true
		}
		rows = append(rows, row)
	}

	if !hasEnabledRow {
		switch options.CommitType {
		case CommitTypePr:
			return []templates.GitLog{}, errors.New("no new commits with PRs")
		case CommitTypeNoPr:
			return []templates.GitLog{}, errors.New("no new commits without PRs")
		case CommitTypeBoth:
			return []templates.GitLog{}, errors.New("no new commits")
		default:
			panic("Unknown commit type " + fmt.Sprint(options.CommitType))
		}
	}

	var footer string
	if hasDuplicates && util.GetUserConfig().ShowDuplicateSubjectLegend {
		countLegendShown(util.LegendDuplicateSubject)
		footer = templates.DuplicateSubjectLegend
	}
	commitSelector := NewCommitSelector(options.Prompt, columns, rows, options.MultiSelect, rowEnabled, footer)
	program := newProgram(commitSelector, appConfig.Io)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go updateRowBranches(ctx, program, newCommits, prBranches, options.GitDir, &wg)
	finalModel := runProgram(appConfig.Io, program)
	cancel()
	wg.Wait()
	selected := finalModel.(CommitSelector).SelectedRows
	slices.Sort(selected)
	selectedCommits := make([]templates.GitLog, 0, len(selected))
	// reverse the selected indexes to do cherry-picks in order.
	for _, selectedRow := range slices.Backward(selected) {
		selectedCommits = append(selectedCommits, newCommits[selectedRow])
	}
	return selectedCommits, nil
}

// Add lines for each commit for every PR that has more than one commit.
// Stops early when done is closed (i.e. when the interactive UI exits).
func updateRowBranches(ctx context.Context, program *tea.Program, newCommits []templates.GitLog, prBranches []string, gitDir string, wg *sync.WaitGroup) {
	defer wg.Done()
	for i, newCommit := range newCommits {
		if ctx.Err() != nil {
			return
		}
		if slices.Contains(prBranches, newCommit.Branch) {
			branchCommits := templates.GetNewCommits(newCommit.Branch, gitDir)
			summary := newCommit.Subject
			if len(branchCommits) > 1 {
				// Reverse the commits so that they are ordered from the earliest to more recent.
				slices.Reverse(branchCommits)
				// Remove the earliest commit as it is equal to newCommit
				branchCommits = branchCommits[1:]
				// Only include the three most recent commits, if there are more than three commits use a "hiding xx" message
				if len(branchCommits) > 3 {
					hidingMessage := hidingColor.Sprint("\n- [hiding ", (len(branchCommits) - 2), " previous...]")
					summary += hidingMessage
					branchCommits = branchCommits[len(branchCommits)-2:]
				}
				for _, branchCommit := range branchCommits {
					summary += "\n- " + branchCommit.Subject
				}
			}
			program.Send(updateCommitSelectorSummaryMsg{Index: i, Summary: summary})
		}
	}
}

// GetBranchSelectionWithFilter displays an interactive selector for branches with optional filtering.
// The rowEnabled function can be used to disable certain rows (branches) from selection.
// If rowEnabled is nil, all rows are enabled.
// Returns an empty array if user cancelled.
func GetBranchSelectionWithFilter(branches []string, prompt string, rowEnabled func(row int) bool) ([]string, error) {
	appConfig := util.GetAppConfig()
	if len(branches) == 0 {
		return []string{}, errors.New("no branches to select from")
	}

	columns := []string{"#", "Branch Name"}
	rows := make([][]string, len(branches))
	for i, branch := range branches {
		indexString := fmt.Sprintf("%d", i+1)
		rows[i] = []string{indexString, branch}
	}

	// If no rowEnabled function provided, all rows are enabled
	if rowEnabled == nil {
		rowEnabled = func(row int) bool {
			return true
		}
	}

	branchSelector := NewCommitSelector(prompt, columns, rows, true, rowEnabled, "")
	program := newProgram(branchSelector, appConfig.Io)
	finalModel := runProgram(appConfig.Io, program)
	selected := finalModel.(CommitSelector).SelectedRows

	if len(selected) == 0 {
		return []string{}, nil
	}

	slices.Sort(selected)
	selectedBranches := make([]string, 0, len(selected))
	for _, selectedRow := range selected {
		if selectedRow >= 0 && selectedRow < len(branches) {
			selectedBranches = append(selectedBranches, branches[selectedRow])
		}
	}

	return selectedBranches, nil
}

// WorktreeOption represents a worktree that can be selected.
type WorktreeOption struct {
	Branch string
	Path   string
}

// GetWorktreeSelection displays an interactive single-select table for worktrees,
// showing branch and directory columns. Returns the selected index, or -1 if cancelled.
func GetWorktreeSelection(worktrees []WorktreeOption, prompt string) (int, error) {
	appConfig := util.GetAppConfig()
	if len(worktrees) == 0 {
		return -1, errors.New("no worktrees to select from")
	}
	columns := []string{"#", "Directory", "Branch"}
	rows := make([][]string, len(worktrees))
	for i, wt := range worktrees {
		rows[i] = []string{fmt.Sprintf("%d", i+1), wt.Path, wt.Branch}
	}
	selector := NewCommitSelector(prompt, columns, rows, false, func(row int) bool { return true }, "")
	program := newProgram(selector, appConfig.Io)
	finalModel := runProgram(appConfig.Io, program)
	selected := finalModel.(CommitSelector).SelectedRows
	if len(selected) == 0 {
		return -1, nil
	}
	slices.Sort(selected)
	return selected[0], nil
}
