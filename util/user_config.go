package util

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"

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
	PromptForReview PromptForReviewType
}

type YamlConfig struct {
	PromptForReview PromptForReviewType `yaml:"promptForReview"`
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
		panic(fmt.Sprint("Could not parse config file: ", err))
	}
	if cfg.PromptForReview != "" && !cfg.PromptForReview.IsValid() {
		panic("invalid promptForReview value in config file: " + string(cfg.PromptForReview))
	}
	return cfg
}

// NewUserConfig merges hardcoded defaults, file config, and --config flag entries.
func NewUserConfig(fileConfig YamlConfig, flagValues map[string]string) UserConfig {
	config := UserConfig{PromptForReview: PromptForReviewPromptN}
	if fileConfig.PromptForReview != "" {
		config.PromptForReview = fileConfig.PromptForReview
	}
	for key, value := range flagValues {
		switch key {
		case "promptForReview":
			v := PromptForReviewType(value)
			if !v.IsValid() {
				panic("invalid promptForReview value: " + value)
			}
			config.PromptForReview = v
		default:
			panic("unknown --config key: " + key)
		}
	}
	return config
}
