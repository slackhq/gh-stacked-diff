package commands

import (
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestNewUserConfig_Default(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil)
	assert.Equal(t, util.PromptForReviewPromptN, config.PromptForReview)
}

func TestNewUserConfig_ValidValues(t *testing.T) {
	for _, value := range []string{"never", "promptY", "promptN"} {
		config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"promptForReview": value})
		assert.Equal(t, util.PromptForReviewType(value), config.PromptForReview)
	}
}

func TestNewUserConfig_InvalidValue(t *testing.T) {
	assert.PanicsWithValue(t, "invalid promptForReview value: invalid", func() {
		util.NewUserConfig(util.YamlConfig{}, map[string]string{"promptForReview": "invalid"})
	})
}

func TestNewUserConfig_UnknownKey(t *testing.T) {
	assert.PanicsWithValue(t, "unknown --config key: foo", func() {
		util.NewUserConfig(util.YamlConfig{}, map[string]string{"foo": "bar"})
	})
}

func TestNewUserConfig_FileConfig(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{PromptForReview: util.PromptForReviewNever}, nil)
	assert.Equal(t, util.PromptForReviewNever, config.PromptForReview)
}

func TestNewUserConfig_FlagOverridesFile(t *testing.T) {
	config := util.NewUserConfig(
		util.YamlConfig{PromptForReview: util.PromptForReviewNever},
		map[string]string{"promptForReview": "promptY"},
	)
	assert.Equal(t, util.PromptForReviewPromptY, config.PromptForReview)
}
