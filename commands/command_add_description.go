/*
Ask Claude to summarize a PR using markdown (write to a file)
then use `gh` to update the PR description
*/
package commands

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/ai"
	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

const GENERATED_CLAUDE_SUMMARY_BEGIN = "\n<details>\n<summary>Claude Generated Summary (click me)</summary>\n\n"
const GENERATED_CLAUDE_SUMMARY_END = "\n\n</details>\n"

func createAddDescriptionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-description [commitIndicator...]",
		Short: "Adds a Claude generated summary to a PR description",
		Long:  "Asks Claude to generate a description for a PR, and then adds the summary to the PR description.\nIf the PR description already has a Claude summary, this replaces it.",
		Args:  cobra.NoArgs,
		Annotations: map[string]string{
			checkRepoAnnotation: "true",
		},
		Hidden: true, // WIP: This command is not yet ready for general use. It requires an AI CLI tool to be configured.
	}
	indicatorTypeString := addIndicatorFlag(cmd)
	cmd.Run = func(cmd *cobra.Command, args []string) {
		selectPrsOptions := interactive.CommitSelectionOptions{
			Prompt:      "What PRs do you want to add AI generated descriptions to?",
			CommitType:  interactive.CommitTypePr,
			MultiSelect: true,
		}
		aiCommand := ai.GetAiCommandInteractive()
		slog.Info(fmt.Sprint("commands", aiCommand))
		targetCommits := getTargetCommits(args, indicatorTypeString, selectPrsOptions)
		executeAddDescription(targetCommits)
	}
	return cmd
}

func executeAddDescription(targetCommits []templates.GitLog) {
	for _, targetCommit := range targetCommits {
		// Note: while possible to parallize, the output is less confusing to do serially.
		addDescriptionForBranch(targetCommit.Branch)
	}
}

func addDescriptionForBranch(branch string) {
	appConfig := util.GetAppConfig()
	slog.Info("Getting PR info for branch " + branch)
	pr := gitutil.GetUnmergedPR(branch)
	if pr == nil {
		panic("No open PR found for branch " + branch)
	}
	slog.Info("Asking Claude to generate PR description")
	description := createAiDescription(*pr)
	newBody := getNewPrBody(*pr, description)
	slog.Info("Adding comment to PR description")
	util.ExecuteOrDie(util.ExecuteOptions{Io: appConfig.Io}, "gh", "pr", "edit", pr.Number, "--body", newBody)
}

func createAiDescription(pr gitutil.PrInfo) string {
	appConfig := util.GetAppConfig()
	prompt := ai.GetPromptPrDescription(pr.Number)
	outWriter := util.NewWriteRecorder(appConfig.Io.Out)
	aiCommand := ai.GetAiCommandInteractive()
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

func getNewPrBody(pr gitutil.PrInfo, description string) string {
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
