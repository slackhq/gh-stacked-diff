package commands

import (
	"testing"
	"time"

	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestNewUserConfig_Default(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil, 0)
	assert.Equal(t, util.PromptForReviewPromptN, config.PromptForReview)
}

func TestNewUserConfig_ValidValues(t *testing.T) {
	for _, value := range []string{"never", "promptY", "promptN"} {
		config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"promptForReview": value}, 0)
		assert.Equal(t, util.PromptForReviewType(value), config.PromptForReview)
	}
}

func TestNewUserConfig_InvalidValue(t *testing.T) {
	assert.PanicsWithValue(t, "invalid promptForReview value: invalid", func() {
		util.NewUserConfig(util.YamlConfig{}, map[string]string{"promptForReview": "invalid"}, 0)
	})
}

func TestNewUserConfig_UnknownKey(t *testing.T) {
	assert.PanicsWithValue(t, "unknown --config key: foo", func() {
		util.NewUserConfig(util.YamlConfig{}, map[string]string{"foo": "bar"}, 0)
	})
}

func TestNewUserConfig_FileConfig(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{PromptForReview: util.PromptForReviewNever}, nil, 0)
	assert.Equal(t, util.PromptForReviewNever, config.PromptForReview)
}

func TestNewUserConfig_FlagOverridesFile(t *testing.T) {
	config := util.NewUserConfig(
		util.YamlConfig{PromptForReview: util.PromptForReviewNever},
		map[string]string{"promptForReview": "promptY"},
		0,
	)
	assert.Equal(t, util.PromptForReviewPromptY, config.PromptForReview)
}

func TestNewUserConfig_DefaultPollInterval(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil, 0)
	assert.Equal(t, util.DefaultPollInterval, config.PollInterval)
}

func TestNewUserConfig_PollIntervalFromFlag(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"pollInterval": "1m"}, 0)
	assert.Equal(t, time.Minute, config.PollInterval)
}

func TestNewUserConfig_PollIntervalFromFile(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{PollInterval: "10s"}, nil, 0)
	assert.Equal(t, 10*time.Second, config.PollInterval)
}

func TestNewUserConfig_PollIntervalFlagOverridesFile(t *testing.T) {
	config := util.NewUserConfig(
		util.YamlConfig{PollInterval: "10s"},
		map[string]string{"pollInterval": "1m"},
		0,
	)
	assert.Equal(t, time.Minute, config.PollInterval)
}

func TestNewUserConfig_InvalidPollInterval(t *testing.T) {
	assert.PanicsWithValue(t, "invalid pollInterval value: notaduration", func() {
		util.NewUserConfig(util.YamlConfig{}, map[string]string{"pollInterval": "notaduration"}, 0)
	})
}

func TestNewUserConfig_TicketUrlPatternDefault(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil, 0)
	assert.Equal(t, "", config.TicketUrlPattern)
}

func TestNewUserConfig_TicketUrlPatternFromFlag(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"ticketUrlPattern": util.ExampleTicketUrlPattern}, 0)
	assert.Equal(t, util.ExampleTicketUrlPattern, config.TicketUrlPattern)
}

func TestNewUserConfig_TicketUrlPatternFromFile(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{TicketUrlPattern: util.ExampleTicketUrlPattern}, nil, 0)
	assert.Equal(t, util.ExampleTicketUrlPattern, config.TicketUrlPattern)
}

func TestNewUserConfig_TicketUrlPatternFlagOverridesFile(t *testing.T) {
	config := util.NewUserConfig(
		util.YamlConfig{TicketUrlPattern: "https://file.example.com/{TicketNumber}"},
		map[string]string{"ticketUrlPattern": "https://flag.example.com/{TicketNumber}"},
		0,
	)
	assert.Equal(t, "https://flag.example.com/{TicketNumber}", config.TicketUrlPattern)
}

func TestNewUserConfig_WorktreeMainBranchGuardDefault(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil, 0)
	assert.Equal(t, util.WorktreeMainBranchGuardPath, config.WorktreeMainBranchGuard)
}

func TestNewUserConfig_WorktreeMainBranchGuardFromFlag(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"worktreeMainBranchGuard": "none"}, 0)
	assert.Equal(t, util.WorktreeMainBranchGuardNone, config.WorktreeMainBranchGuard)
}

func TestNewUserConfig_WorktreeMainBranchGuardFromFile(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{WorktreeMainBranchGuard: util.WorktreeMainBranchGuardNone}, nil, 0)
	assert.Equal(t, util.WorktreeMainBranchGuardNone, config.WorktreeMainBranchGuard)
}

func TestNewUserConfig_WorktreeMainBranchGuardFlagOverridesFile(t *testing.T) {
	config := util.NewUserConfig(
		util.YamlConfig{WorktreeMainBranchGuard: util.WorktreeMainBranchGuardNone},
		map[string]string{"worktreeMainBranchGuard": "path"},
		0,
	)
	assert.Equal(t, util.WorktreeMainBranchGuardPath, config.WorktreeMainBranchGuard)
}

func TestNewUserConfig_WorktreeMainBranchGuardInvalidValue(t *testing.T) {
	assert.PanicsWithValue(t, "invalid worktreeMainBranchGuard value: invalid", func() {
		util.NewUserConfig(util.YamlConfig{}, map[string]string{"worktreeMainBranchGuard": "invalid"}, 0)
	})
}

func TestNewUserConfig_ShowWorktreesDefault(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil, 0)
	assert.Equal(t, true, config.ShowWorktrees)
}

func TestNewUserConfig_ShowWorktreesFromFlag(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"showWorktrees": "false"}, 0)
	assert.Equal(t, false, config.ShowWorktrees)
}

func TestNewUserConfig_ShowWorktreesFromFile(t *testing.T) {
	showWorktrees := false
	config := util.NewUserConfig(util.YamlConfig{ShowWorktrees: &showWorktrees}, nil, 0)
	assert.Equal(t, false, config.ShowWorktrees)
}

func TestNewUserConfig_ShowWorktreesFlagOverridesFile(t *testing.T) {
	showWorktrees := true
	config := util.NewUserConfig(
		util.YamlConfig{ShowWorktrees: &showWorktrees},
		map[string]string{"showWorktrees": "false"},
		0,
	)
	assert.Equal(t, false, config.ShowWorktrees)
}

func TestNewUserConfig_ShowWorktreesInvalidValue(t *testing.T) {
	assert.PanicsWithValue(t, "invalid showWorktrees value: invalid (must be true or false)", func() {
		util.NewUserConfig(util.YamlConfig{}, map[string]string{"showWorktrees": "invalid"}, 0)
	})
}

func TestNewUserConfig_ShowUiLegendDefault(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil, 0)
	assert.Equal(t, true, config.ShowUiLegend)
}

func TestNewUserConfig_ShowUiLegendFromFlag(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"showUiLegend": "false"}, 0)
	assert.Equal(t, false, config.ShowUiLegend)
}

func TestNewUserConfig_ShowUiLegendFromFile(t *testing.T) {
	showUiLegend := false
	config := util.NewUserConfig(util.YamlConfig{ShowUiLegend: &showUiLegend}, nil, 0)
	assert.Equal(t, false, config.ShowUiLegend)
}

func TestNewUserConfig_ShowUiLegendFlagOverridesFile(t *testing.T) {
	showUiLegend := true
	config := util.NewUserConfig(
		util.YamlConfig{ShowUiLegend: &showUiLegend},
		map[string]string{"showUiLegend": "false"},
		0,
	)
	assert.Equal(t, false, config.ShowUiLegend)
}

func TestNewUserConfig_ShowUiLegendInvalidValue(t *testing.T) {
	assert.PanicsWithValue(t, "invalid showUiLegend value: invalid (must be true or false)", func() {
		util.NewUserConfig(util.YamlConfig{}, map[string]string{"showUiLegend": "invalid"}, 0)
	})
}

func TestNewUserConfig_ShowUiLegendDefaultFalseWhenShownCountReachesMax(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil, util.MaxUiLegendShownCount)
	assert.Equal(t, false, config.ShowUiLegend)
}

func TestNewUserConfig_ShowUiLegendDefaultTrueWhenShownCountBelowMax(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, nil, util.MaxUiLegendShownCount-1)
	assert.Equal(t, true, config.ShowUiLegend)
}

func TestNewUserConfig_ShowUiLegendExplicitTrueOverridesShownCount(t *testing.T) {
	showUiLegend := true
	config := util.NewUserConfig(util.YamlConfig{ShowUiLegend: &showUiLegend}, nil, util.MaxUiLegendShownCount+2)
	assert.Equal(t, true, config.ShowUiLegend)
}

func TestNewUserConfig_ShowUiLegendFlagOverridesShownCount(t *testing.T) {
	config := util.NewUserConfig(util.YamlConfig{}, map[string]string{"showUiLegend": "true"}, util.MaxUiLegendShownCount+2)
	assert.Equal(t, true, config.ShowUiLegend)
}

func TestShowUiLegend_ShownCountStopsAtMax(t *testing.T) {
	tempDir := t.TempDir()
	util.SetAppConfig(util.AppConfig{ConfigHome: tempDir})

	shownCount := 0
	// Simulate more runs than MaxUiLegendShownCount to verify the count stops incrementing.
	for range util.MaxUiLegendShownCount + 2 {
		metrics := util.LoadMetricsFile()
		config := util.NewUserConfig(util.YamlConfig{}, nil, metrics.UiLegendShownCount)
		if config.ShowUiLegend {
			shownCount++
			util.IncrementUiLegendShownCount()
		}
	}

	assert.Equal(t, util.MaxUiLegendShownCount, shownCount)
	metrics := util.LoadMetricsFile()
	assert.Equal(t, util.MaxUiLegendShownCount, metrics.UiLegendShownCount)
}
