package util

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type MetricsConfig struct {
	UiLegendShownCount int `yaml:"uiLegendShownCount,omitempty"`
}

// LoadMetricsFile reads metrics.yaml from ConfigHome if it exists.
func LoadMetricsFile() MetricsConfig {
	configFile := GetConfigFile("metrics.yaml")
	if configFile == "" {
		return MetricsConfig{}
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		panic(fmt.Sprint("Could not read metrics file: ", err))
	}
	var cfg MetricsConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		panic(fmt.Sprint("Could not parse metrics file: ", err))
	}
	return cfg
}

// IncrementUiLegendShownCount increments the uiLegendShownCount in metrics.yaml.
func IncrementUiLegendShownCount() {
	metrics := LoadMetricsFile()
	metrics.UiLegendShownCount++
	metricsPath := filepath.Join(GetAppConfig().ConfigHome, "metrics.yaml")
	data, err := yaml.Marshal(metrics)
	if err != nil {
		panic(fmt.Sprint("Could not marshal metrics: ", err))
	}
	if err := os.WriteFile(metricsPath, data, 0644); err != nil {
		panic(fmt.Sprint("Could not write metrics file: ", err))
	}
}
