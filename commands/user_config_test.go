package commands

import (
	"testing"
	"time"

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

func TestNewUserConfig_DefaultPollInterval(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil)
	assert.Equal(t, util.DefaultPollInterval, config.PollInterval)
}

func TestNewUserConfig_PollIntervalFromFlag(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"pollInterval": "1m"})
	assert.Equal(t, time.Minute, config.PollInterval)
}

func TestNewUserConfig_PollIntervalFromFile(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{PollInterval: "10s"}, nil)
	assert.Equal(t, 10*time.Second, config.PollInterval)
}

func TestNewUserConfig_PollIntervalFlagOverridesFile(t *testing.T) {
	config := util.NewUserConfig(
		util.YamlConfig{PollInterval: "10s"},
		map[string]string{"pollInterval": "1m"},
	)
	assert.Equal(t, time.Minute, config.PollInterval)
}

func TestNewUserConfig_InvalidPollInterval(t *testing.T) {
	assert.PanicsWithValue(t, "invalid pollInterval value: notaduration", func() {
		util.NewUserConfig(util.YamlConfig{}, map[string]string{"pollInterval": "notaduration"})
	})
}

func TestNewUserConfig_TicketUrlPatternDefault(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil)
	assert.Equal(t, "", config.TicketUrlPattern)
}

func TestNewUserConfig_TicketUrlPatternFromFlag(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"ticketUrlPattern": util.ExampleTicketUrlPattern})
	assert.Equal(t, util.ExampleTicketUrlPattern, config.TicketUrlPattern)
}

func TestNewUserConfig_TicketUrlPatternFromFile(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{TicketUrlPattern: util.ExampleTicketUrlPattern}, nil)
	assert.Equal(t, util.ExampleTicketUrlPattern, config.TicketUrlPattern)
}

func TestNewUserConfig_TicketUrlPatternFlagOverridesFile(t *testing.T) {
	config := util.NewUserConfig(
		util.YamlConfig{TicketUrlPattern: "https://file.example.com/{TicketNumber}"},
		map[string]string{"ticketUrlPattern": "https://flag.example.com/{TicketNumber}"},
	)
	assert.Equal(t, "https://flag.example.com/{TicketNumber}", config.TicketUrlPattern)
}

func TestNewUserConfig_WorktreeMainBranchGuardDefault(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil)
	assert.Equal(t, util.WorktreeMainBranchGuardPath, config.WorktreeMainBranchGuard)
}

func TestNewUserConfig_WorktreeMainBranchGuardFromFlag(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"worktreeMainBranchGuard": "none"})
	assert.Equal(t, util.WorktreeMainBranchGuardNone, config.WorktreeMainBranchGuard)
}

func TestNewUserConfig_WorktreeMainBranchGuardFromFile(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{WorktreeMainBranchGuard: util.WorktreeMainBranchGuardNone}, nil)
	assert.Equal(t, util.WorktreeMainBranchGuardNone, config.WorktreeMainBranchGuard)
}

func TestNewUserConfig_WorktreeMainBranchGuardFlagOverridesFile(t *testing.T) {
	config := util.NewUserConfig(
		util.YamlConfig{WorktreeMainBranchGuard: util.WorktreeMainBranchGuardNone},
		map[string]string{"worktreeMainBranchGuard": "path"},
	)
	assert.Equal(t, util.WorktreeMainBranchGuardPath, config.WorktreeMainBranchGuard)
}

func TestNewUserConfig_WorktreeMainBranchGuardInvalidValue(t *testing.T) {
	assert.PanicsWithValue(t, "invalid worktreeMainBranchGuard value: invalid", func() {
		util.NewUserConfig(util.YamlConfig{}, map[string]string{"worktreeMainBranchGuard": "invalid"})
	})
}

func TestNewUserConfig_ShowWorktreesDefault(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil)
	assert.Equal(t, true, config.ShowWorktrees)
}

func TestNewUserConfig_ShowWorktreesFromFlag(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"showWorktrees": "false"})
	assert.Equal(t, false, config.ShowWorktrees)
}

func TestNewUserConfig_ShowWorktreesFromFile(t *testing.T) {
	showWorktrees := false
	config := util.NewUserConfig(util.YamlConfig{ShowWorktrees: &showWorktrees}, nil)
	assert.Equal(t, false, config.ShowWorktrees)
}

func TestNewUserConfig_ShowWorktreesFlagOverridesFile(t *testing.T) {
	showWorktrees := true
	config := util.NewUserConfig(
		util.YamlConfig{ShowWorktrees: &showWorktrees},
		map[string]string{"showWorktrees": "false"},
	)
	assert.Equal(t, false, config.ShowWorktrees)
}

func TestNewUserConfig_ShowWorktreesInvalidValue(t *testing.T) {
	assert.PanicsWithValue(t, "invalid showWorktrees value: invalid (must be true or false)", func() {
		util.NewUserConfig(util.YamlConfig{}, map[string]string{"showWorktrees": "invalid"})
	})
}
