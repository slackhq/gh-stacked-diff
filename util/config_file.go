package util

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// GetConfigFile returns the full path to a config file in ConfigHome
// if it exists, or "" if it does not.
func GetConfigFile(filenameWithoutPath string) string {
	fullPath := filepath.Join(GetAppConfig().ConfigHome, filenameWithoutPath)
	_, err := os.Stat(fullPath)
	if err == nil {
		return fullPath
	}
	if !errors.Is(err, os.ErrNotExist) {
		slog.Warn(fmt.Sprint("Could not stat config file ", fullPath, ": ", err))
	}
	return ""
}
