package util

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LegendType identifies a specific interactive UI legend.
type LegendType string

const (
	LegendUserSelection       LegendType = "userSelection"
	LegendTableSelection      LegendType = "tableSelection"
	LegendTableMultiselection LegendType = "tableMultiselection"
	LegendDuplicateSubject    LegendType = "duplicateSubject"
)

type MetricsConfig struct {
	UserSelectionLegendShownCount       int `yaml:"userSelectionLegendShownCount,omitempty"`
	TableSelectionLegendShownCount      int `yaml:"tableSelectionLegendShownCount,omitempty"`
	TableMultiselectionLegendShownCount int `yaml:"tableMultiselectionLegendShownCount,omitempty"`
	DuplicateSubjectLegendShownCount    int `yaml:"duplicateSubjectLegendShownCount,omitempty"`
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
	if err := decoder.Decode(&cfg); err != nil {
		panic(fmt.Sprint("Could not parse metrics file: ", err))
	}
	return cfg
}

// GetLegendShownCount returns the shown count for the given legend type.
func (m MetricsConfig) GetLegendShownCount(legend LegendType) int {
	switch legend {
	case LegendUserSelection:
		return m.UserSelectionLegendShownCount
	case LegendTableSelection:
		return m.TableSelectionLegendShownCount
	case LegendTableMultiselection:
		return m.TableMultiselectionLegendShownCount
	case LegendDuplicateSubject:
		return m.DuplicateSubjectLegendShownCount
	default:
		panic(fmt.Sprint("unknown legend type: ", legend))
	}
}

// IncrementLegendShownCount increments the shown count for the given legend type in metrics.yaml.
func IncrementLegendShownCount(legend LegendType) {
	metrics := LoadMetricsFile()
	switch legend {
	case LegendUserSelection:
		metrics.UserSelectionLegendShownCount++
	case LegendTableSelection:
		metrics.TableSelectionLegendShownCount++
	case LegendTableMultiselection:
		metrics.TableMultiselectionLegendShownCount++
	case LegendDuplicateSubject:
		metrics.DuplicateSubjectLegendShownCount++
	default:
		panic(fmt.Sprint("unknown legend type: ", legend))
	}
	metricsPath := filepath.Join(GetAppConfig().ConfigHome, "metrics.yaml")
	data, err := yaml.Marshal(metrics)
	if err != nil {
		panic(fmt.Sprint("Could not marshal metrics: ", err))
	}
	if err := os.WriteFile(metricsPath, data, 0644); err != nil {
		panic(fmt.Sprint("Could not write metrics file: ", err))
	}
}
