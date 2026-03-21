package commands

import (
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// UserConfig holds runtime configuration from --config flag key=value entries.
type UserConfig struct {
	promptForReview util.PromptForReviewType
}

// NewUserConfig parses --config key=value entries and returns a UserConfig.
func NewUserConfig(configValues []string) UserConfig {
	config := UserConfig{promptForReview: util.PromptForReviewPromptN}
	for _, entry := range configValues {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			panic("invalid --config entry, expected key=value: " + entry)
		}
		switch key {
		case "promptForReview":
			v := util.PromptForReviewType(value)
			if !v.IsValid() {
				panic("invalid promptForReview value: " + value)
			}
			config.promptForReview = v
		default:
			panic("unknown --config key: " + key)
		}
	}
	return config
}

// PromptForReview returns the configured prompt-for-review behavior.
func (c UserConfig) PromptForReview() util.PromptForReviewType {
	return c.promptForReview
}
