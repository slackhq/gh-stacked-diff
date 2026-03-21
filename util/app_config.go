package util

import (
	"io"
)

// Allows unit testing the use of standard i/o.
type StdIo struct {
	Out io.Writer
	Err io.Writer
	In  io.Reader
}

// Config to help with unit testing the app.
// For example, it allows testing code paths that would otherwise call os.Exit().
type AppConfig struct {
	Io            StdIo
	AppExecutable string         // Path of this executable.
	Exit          func(code int) // Call os.Exit with the given code, or panic during unit tests.
	UserCacheDir  string         // os.UserCacheDir or a dir specific for each test in unit tests.
	ConfigHome    string         // Path to ~/.gh-stacked-diff/ or a test-specific dir in unit tests.
	DemoMode      bool
}

var globalAppConfig AppConfig

// SetAppConfig sets the global AppConfig. Must be called once at startup (main or test setup).
func SetAppConfig(config AppConfig) {
	globalAppConfig = config
}

// GetAppConfig returns the global AppConfig.
func GetAppConfig() AppConfig {
	return globalAppConfig
}
