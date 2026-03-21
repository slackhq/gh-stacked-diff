package util

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

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

// UserConfig holds runtime configuration from config file and --config flag key=value entries.
type UserConfig struct {
	promptForReview PromptForReviewType
}

// PromptForReview returns the configured prompt-for-review behavior.
func (c UserConfig) PromptForReview() PromptForReviewType {
	return c.promptForReview
}

type YamlConfig struct {
	PromptForReview PromptForReviewType `yaml:"promptForReview"`
}

// LoadUserConfigFile reads ~/.gh-stacked-diff/config.yaml if it exists.
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
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		panic(fmt.Sprint("Could not parse config file: ", err))
	}
	if cfg.PromptForReview != "" && !cfg.PromptForReview.IsValid() {
		panic("invalid promptForReview value in config file: " + string(cfg.PromptForReview))
	}
	return cfg
}

// NewUserConfig merges hardcoded defaults, file config, and --config flag entries.
func NewUserConfig(fileConfig YamlConfig, flagValues []string) UserConfig {
	config := UserConfig{promptForReview: PromptForReviewPromptN}
	if fileConfig.PromptForReview != "" {
		config.promptForReview = fileConfig.PromptForReview
	}
	for _, entry := range flagValues {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			panic("invalid --config entry, expected key=value: " + entry)
		}
		switch key {
		case "promptForReview":
			v := PromptForReviewType(value)
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
