package util

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// GetConfigFile returns the full path to a config file in ~/.gh-stacked-diff/
// if it exists, or "" if it does not.
func GetConfigFile(filenameWithoutPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprint("Could not get home dir", err))
	}
	fullPath := filepath.Join(home, ".gh-stacked-diff", filenameWithoutPath)
	_, err = os.Stat(fullPath)
	if err == nil {
		return fullPath
	}
	if !errors.Is(err, os.ErrNotExist) {
		slog.Warn(fmt.Sprint("Could not stat config file ", fullPath, ": ", err))
	}
	return ""
}
