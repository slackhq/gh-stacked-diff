package commands

import (
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestNewUserConfig_Default(t *testing.T) {
	config := NewUserConfig(yamlConfig{}, nil)
	assert.Equal(t, util.PromptForReviewPromptN, config.PromptForReview())
}

func TestNewUserConfig_ValidValues(t *testing.T) {
	for _, value := range []string{"never", "promptY", "promptN"} {
		config := NewUserConfig(yamlConfig{}, []string{"promptForReview=" + value})
		assert.Equal(t, util.PromptForReviewType(value), config.PromptForReview())
	}
}

func TestNewUserConfig_InvalidValue(t *testing.T) {
	assert.PanicsWithValue(t, "invalid promptForReview value: invalid", func() {
		NewUserConfig(yamlConfig{}, []string{"promptForReview=invalid"})
	})
}

func TestNewUserConfig_UnknownKey(t *testing.T) {
	assert.PanicsWithValue(t, "unknown --config key: foo", func() {
		NewUserConfig(yamlConfig{}, []string{"foo=bar"})
	})
}

func TestNewUserConfig_MissingEquals(t *testing.T) {
	assert.PanicsWithValue(t, "invalid --config entry, expected key=value: noequals", func() {
		NewUserConfig(yamlConfig{}, []string{"noequals"})
	})
}

func TestNewUserConfig_FileConfig(t *testing.T) {
	config := NewUserConfig(yamlConfig{PromptForReview: util.PromptForReviewNever}, nil)
	assert.Equal(t, util.PromptForReviewNever, config.PromptForReview())
}

func TestNewUserConfig_FlagOverridesFile(t *testing.T) {
	config := NewUserConfig(
		yamlConfig{PromptForReview: util.PromptForReviewNever},
		[]string{"promptForReview=promptY"},
	)
	assert.Equal(t, util.PromptForReviewPromptY, config.PromptForReview())
}
