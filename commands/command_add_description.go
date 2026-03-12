/*
Ask Claude to summarize a PR using markdown (write to a file)
then use `gh` to update the PR description
*/
package commands

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/ai"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

const GENERATED_CLAUDE_SUMMARY_BEGIN = "\n<details>\n<summary>Claude Generated Summary (click me)</summary>\n\n"
const GENERATED_CLAUDE_SUMMARY_END = "\n\n</details>\n"

func createAddDescriptionCommand() Command {
	flagSet := flag.NewFlagSet("add-description", flag.ContinueOnError)
	indicatorTypeString := addIndicatorFlag(flagSet)
	return Command{
		FlagSet:         flagSet,
		DefaultLogLevel: slog.LevelInfo,
		Summary:         "Adds a Claude generated summary to a PR description",
		Description: `Asks Claude to generate a description for a PR, and then adds the summary to the PR description. 
If the PR description already has a Claude summary, this replaces it.`,
		Usage:         "sd " + flagSet.Name() + " [flags] [commitIndicator]",
		SkipRepoCheck: false,
		OnSelected: func(appConfig util.AppConfig, command Command) {
			if flagSet.NArg() > 0 {
				commandError(appConfig, flagSet, "unexpected arguments", command.Usage)
			}
			selectPrsOptions := interactive.CommitSelectionOptions{
				Prompt:      "What PRs do you want to add AI generated descriptions to?",
				CommitType:  interactive.CommitTypePr,
				MultiSelect: true,
			}
			aiCommand := ai.GetAiCommandInteractive(appConfig)
			slog.Info(fmt.Sprint("commands", aiCommand))
			targetCommits := getTargetCommits(appConfig, command, flagSet.Args(), indicatorTypeString, selectPrsOptions)
			executeAddDescription(appConfig, targetCommits)
		},
		Hidden: true, // WIP: This command is not yet ready for general use. It requires an AI CLI tool to be configured.
	}
}

func executeAddDescription(appConfig util.AppConfig, targetCommits []templates.GitLog) {
	for _, targetCommit := range targetCommits {
		// Note: while possible to parallize, the output is less confusing to do serially.
		addDescriptionForBranch(appConfig, targetCommit.Branch)
	}
}

func addDescriptionForBranch(appConfig util.AppConfig, branch string) {
	slog.Info("Getting PR info for branch " + branch)
	pr := util.GetUnmergedPR(branch)
	if pr == nil {
		panic("No open PR found for branch " + branch)
	}
	slog.Info("Asking Claude to generate PR description")
	description := createAiDescription(appConfig, *pr)
	newBody := getNewPrBody(*pr, description)
	slog.Info("Adding comment to PR description")
	setPrBody(appConfig, *pr, newBody)
}
func createAiDescription(appConfig util.AppConfig, pr util.PrInfo) string {
	// tmpFile := filepath.Join("/tmp", "gh-stacked-diff-pr-description-"+pr.Number+".md")
	prompt := ai.GetPromptPrDescription(pr.Number)
	outWriter := util.NewWriteRecorder(appConfig.Io.Out)
	aiCommand := ai.GetAiCommandInteractive(appConfig)
	var allArgs = append(aiCommand[1:], "-p", prompt)
	util.ExecuteOrDie(util.ExecuteOptions{Io: util.StdIo{Out: outWriter, Err: appConfig.Io.Err, In: appConfig.Io.In}}, aiCommand[0], allArgs...)
	out := outWriter.String()
	return parseDescription(out)
}

func parseDescription(aiOutput string) string {
	begins := []string{"\n```markdown\n", "\n## Summary\n"}
	ends := []string{"\n```\n", "\n---\n", "\n-\n"}
	start := -1
	beginIndex := -1
	for i, begin := range begins {
		start = strings.Index(aiOutput, begin)
		if start != -1 {
			beginIndex = i
			break
		}
	}
	if start == -1 {
		slog.Warn("Missing markdown start (" + fmt.Sprint(begins) + ") in claude output")
		start = 0
	}
	end := -1
	for _, nextEnd := range ends {
		end = strings.Index(aiOutput[start:], nextEnd)
		if end != -1 {
			break
		}
	}
	if end == -1 {
		slog.Debug("Missing markdown end (" + fmt.Sprint(ends) + ") in claude output")
		end = len(aiOutput[start:])
	}
	return string(aiOutput[start+len(begins[beginIndex]) : start+end])
}

func getNewPrBody(pr util.PrInfo, description string) string {
	bodyComment := GENERATED_CLAUDE_SUMMARY_BEGIN + description + GENERATED_CLAUDE_SUMMARY_END

	bodyWithoutComment := pr.Body
	existingBeginComment := strings.Index(pr.Body, GENERATED_CLAUDE_SUMMARY_BEGIN)
	if existingBeginComment != -1 {
		existingEndComment := strings.Index(pr.Body[existingBeginComment:], GENERATED_CLAUDE_SUMMARY_END)
		if existingEndComment != -1 {
			bodyWithoutComment = pr.Body[0:existingBeginComment] + pr.Body[existingBeginComment+existingEndComment+len(GENERATED_CLAUDE_SUMMARY_END):]
		}
	}

	// Insert description before the first "#### Ticket" if there is one, or at the end.
	newBody := strings.Replace(bodyWithoutComment, "\n#### Ticket", bodyComment+"\n#### Ticket", 1)
	if newBody == bodyWithoutComment {
		newBody = bodyWithoutComment + bodyComment
	}
	return newBody
}

func setPrBody(appConfig util.AppConfig, pr util.PrInfo, newBody string) {
	util.ExecuteOrDie(util.ExecuteOptions{Io: appConfig.Io}, "gh", "pr", "edit", pr.Number, "--body", newBody)
}
