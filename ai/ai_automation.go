package ai

import (
	_ "embed"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/templates"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

type prData struct {
	PrNumber string
}

var aiCommandHistory = util.NewHistoricalData("ai_command.config", -1)

//go:embed templates/ai_prompt_pr_description.template
var aiPromptPrDescription string

//go:embed templates/ai_prompt_pr_review.template
var aiPromptPrReview string

func GetAiCommandInteractive() []string {
	commandAndArgs := aiCommandHistory.ReadHistory()
	if len(commandAndArgs) == 0 {
		const commandInteractivePrompt string = "What is the command to use to launch your AI CLI?"
		commandInteractiveSuggestions := []string{"claude"}
		prompt := interactive.PromptForStringOrDie(commandInteractivePrompt, "", commandInteractiveSuggestions)
		commandAndArgs = strings.Fields(prompt)
		aiCommandHistory.SetHistory(commandAndArgs)
	}
	return commandAndArgs
}

func GetPromptPrDescription(prNumber string) string {
	data := prData{
		PrNumber: prNumber,
	}
	return templates.RunTemplate("ai_prompt_pr_description.template", aiPromptPrDescription, data)
}

func GetPromptPrReview(prNumber string) string {
	data := prData{
		PrNumber: prNumber,
	}
	return templates.RunTemplate("ai_prompt_pr_review.template", aiPromptPrReview, data)
}
