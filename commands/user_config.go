package commands

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
	"gopkg.in/yaml.v3"
)

// UserConfig holds runtime configuration from config file and --config flag key=value entries.
type UserConfig struct {
	promptForReview util.PromptForReviewType
}

type yamlConfig struct {
	PromptForReview util.PromptForReviewType `yaml:"promptForReview"`
}

// loadUserConfigFile reads ~/.gh-stacked-diff/config.yaml if it exists.
func loadUserConfigFile() yamlConfig {
	configFile := util.GetConfigFile("config.yaml")
	if configFile == "" {
		return yamlConfig{}
	}
	slog.Debug(fmt.Sprint("Loading config file: ", configFile))
	data, err := os.ReadFile(configFile)
	if err != nil {
		panic(fmt.Sprint("Could not read config file: ", err))
	}
	var cfg yamlConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		panic(fmt.Sprint("Could not parse config file: ", err))
	}
	if cfg.PromptForReview != "" && !cfg.PromptForReview.IsValid() {
		panic("invalid promptForReview value in config file: " + string(cfg.PromptForReview))
	}
	return cfg
}

// NewUserConfig merges hardcoded defaults, file config, and --config flag entries.
func NewUserConfig(fileConfig yamlConfig, flagValues []string) UserConfig {
	config := UserConfig{promptForReview: util.PromptForReviewPromptN}
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
