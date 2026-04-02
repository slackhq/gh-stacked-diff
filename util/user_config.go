package util

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultPollInterval = 30 * time.Second

// ExampleTicketUrlPattern is the example ticket URL pattern shown in help text, prompts, and tests.
const ExampleTicketUrlPattern = "https://jira.example.com/browse/{TicketNumber}"

// PromptForReviewType controls whether and how the user is prompted to mark a PR as ready for review.
type PromptForReviewType string

const (
	PromptForReviewNever   PromptForReviewType = "never"
	PromptForReviewPromptY PromptForReviewType = "promptY"
	PromptForReviewPromptN PromptForReviewType = "promptN"
)

func (t PromptForReviewType) IsValid() bool {
	switch t {
	case PromptForReviewNever, PromptForReviewPromptY, PromptForReviewPromptN:
		return true
	}
	return false
}

// WorktreeMainBranchGuardType controls how the main branch is determined in secondary worktrees.
type WorktreeMainBranchGuardType string

const (
	WorktreeMainBranchGuardPath WorktreeMainBranchGuardType = "path"
	WorktreeMainBranchGuardNone WorktreeMainBranchGuardType = "none"
)

func (t WorktreeMainBranchGuardType) IsValid() bool {
	switch t {
	case WorktreeMainBranchGuardPath, WorktreeMainBranchGuardNone:
		return true
	}
	return false
}

// UserConfig holds runtime configuration from config file and --config flag key=value entries.
type UserConfig struct {
	PromptForReview         PromptForReviewType
	PollInterval            time.Duration
	TicketUrlPattern        string
	WorktreeMainBranchGuard WorktreeMainBranchGuardType
	ShowWorktrees           bool
}

type YamlConfig struct {
	PromptForReview         PromptForReviewType         `yaml:"promptForReview,omitempty"`
	PollInterval            string                      `yaml:"pollInterval,omitempty"`
	TicketUrlPattern        string                      `yaml:"ticketUrlPattern,omitempty"`
	WorktreeMainBranchGuard WorktreeMainBranchGuardType `yaml:"worktreeMainBranchGuard,omitempty"`
	ShowWorktrees           *bool                       `yaml:"showWorktrees,omitempty"`
}

// LoadUserConfigFile reads config.yaml from ConfigHome if it exists.
func LoadUserConfigFile() YamlConfig {
	configFile := GetConfigFile("config.yaml")
	if configFile == "" {
		return YamlConfig{}
	}
	slog.Debug(fmt.Sprint("Loading config file: ", configFile))
	data, err := os.ReadFile(configFile)
	if err != nil {
		panic(fmt.Sprint("Could not read config file: ", err))
	}
	var cfg YamlConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		panic(fmt.Sprint("Could not parse config file (", configFile, "): ", err))
	}
	if cfg.PromptForReview != "" && !cfg.PromptForReview.IsValid() {
		panic("invalid promptForReview value in config file: " + string(cfg.PromptForReview))
	}
	if cfg.PollInterval != "" {
		if _, err := time.ParseDuration(cfg.PollInterval); err != nil {
			panic("invalid pollInterval value in config file: " + cfg.PollInterval)
		}
	}
	if cfg.WorktreeMainBranchGuard != "" && !cfg.WorktreeMainBranchGuard.IsValid() {
		panic("invalid worktreeMainBranchGuard value in config file: " + string(cfg.WorktreeMainBranchGuard))
	}
	return cfg
}

// NewUserConfig merges hardcoded defaults, file config, and --config flag entries.
func NewUserConfig(fileConfig YamlConfig, flagValues map[string]string) UserConfig {
	config := UserConfig{PromptForReview: PromptForReviewPromptN, PollInterval: DefaultPollInterval, WorktreeMainBranchGuard: WorktreeMainBranchGuardPath, ShowWorktrees: true}
	if fileConfig.PromptForReview != "" {
		config.PromptForReview = fileConfig.PromptForReview
	}
	if fileConfig.PollInterval != "" {
		d, _ := time.ParseDuration(fileConfig.PollInterval)
		config.PollInterval = d
	}
	if fileConfig.TicketUrlPattern != "" {
		config.TicketUrlPattern = fileConfig.TicketUrlPattern
	}
	if fileConfig.WorktreeMainBranchGuard != "" {
		config.WorktreeMainBranchGuard = fileConfig.WorktreeMainBranchGuard
	}
	if fileConfig.ShowWorktrees != nil {
		config.ShowWorktrees = *fileConfig.ShowWorktrees
	}
	for key, value := range flagValues {
		switch key {
		case "promptForReview":
			v := PromptForReviewType(value)
			if !v.IsValid() {
				panic("invalid promptForReview value: " + value)
			}
			config.PromptForReview = v
		case "pollInterval":
			d, err := time.ParseDuration(value)
			if err != nil {
				panic("invalid pollInterval value: " + value)
			}
			config.PollInterval = d
		case "ticketUrlPattern":
			config.TicketUrlPattern = value
		case "worktreeMainBranchGuard":
			v := WorktreeMainBranchGuardType(value)
			if !v.IsValid() {
				panic("invalid worktreeMainBranchGuard value: " + value)
			}
			config.WorktreeMainBranchGuard = v
		case "showWorktrees":
			switch value {
			case "true":
				config.ShowWorktrees = true
			case "false":
				config.ShowWorktrees = false
			default:
				panic("invalid showWorktrees value: " + value + " (must be true or false)")
			}
		default:
			panic("unknown --config key: " + key)
		}
	}
	return config
}

var globalUserConfig *UserConfig

// SetUserConfig sets the global UserConfig. Should be called early in command execution.
func SetUserConfig(config UserConfig) {
	globalUserConfig = &config
}

// GetUserConfig returns the global UserConfig. Panics if SetUserConfig has not been called.
func GetUserConfig() UserConfig {
	if globalUserConfig == nil {
		panic("GetUserConfig called before SetUserConfig")
	}
	return *globalUserConfig
}

// SaveTicketUrlPattern saves the ticketUrlPattern value to the config file,
// preserving any existing config values.
func SaveTicketUrlPattern(pattern string) {
	fileConfig := LoadUserConfigFile()
	fileConfig.TicketUrlPattern = pattern
	configPath := filepath.Join(GetAppConfig().ConfigHome, "config.yaml")
	data, err := yaml.Marshal(fileConfig)
	if err != nil {
		panic(fmt.Sprint("Could not marshal config: ", err))
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		panic(fmt.Sprint("Could not write config file: ", err))
	}
	slog.Info(fmt.Sprint("Saved ticketUrlPattern to ", configPath))
}
