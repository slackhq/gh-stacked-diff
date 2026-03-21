package util

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetConfigFile returns the full path to a config file in ~/.gh-stacked-diff/
// if it exists, or nil if it does not.
func GetConfigFile(filenameWithoutPath string) *string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprint("Could not get home dir", err))
	}
	fullPath := filepath.Join(home, ".gh-stacked-diff", filenameWithoutPath)
	if _, err := os.Stat(fullPath); err == nil {
		return &fullPath
	}
	return nil
}
