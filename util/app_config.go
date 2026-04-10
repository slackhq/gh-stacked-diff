package util

import (
	"fmt"
	"io"
	"os"
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
	configHome    string         // Path to ~/.gh-stacked-diff/ or a test-specific dir in unit tests.
	DemoMode      bool
}

// NewAppConfig creates a new AppConfig, ensuring the configHome directory exists.
func NewAppConfig(io StdIo, appExecutable string, exit func(code int), userCacheDir string, configHome string, demoMode bool) AppConfig {
	if err := os.MkdirAll(configHome, 0700); err != nil {
		panic(fmt.Sprint("Could not create config directory: ", err))
	}
	return AppConfig{
		Io:            io,
		AppExecutable: appExecutable,
		Exit:          exit,
		UserCacheDir:  userCacheDir,
		configHome:    configHome,
		DemoMode:      demoMode,
	}
}

// ConfigHome returns the path to the config directory.
func (c AppConfig) ConfigHome() string {
	return c.configHome
}

var globalAppConfig *AppConfig

// SetAppConfig sets the global AppConfig. Must be called once at startup (main or test setup).
func SetAppConfig(config AppConfig) {
	globalAppConfig = &config
}

// GetAppConfig returns the global AppConfig. Panics if SetAppConfig has not been called.
func GetAppConfig() AppConfig {
	if globalAppConfig == nil {
		panic("GetAppConfig called before SetAppConfig")
	}
	return *globalAppConfig
}
