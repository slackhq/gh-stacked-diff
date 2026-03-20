/*
Stacked Diff Workflow Command Line Interface

Usage: sd [flags] <command> [args]

Use "sd --help" for a full list of commands and flags.
Use "sd <command> --help" for help on a specific command.
*/
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/commands"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func main() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		panic("Cannot find user cache dir: " + err.Error())
	}
	demoModeEnv, _ := os.LookupEnv("GH_STACKED_DIFF_DEMO_MODE")
	appConfig := util.AppConfig{
		Io:            util.StdIo{Out: os.Stdout, Err: os.Stderr, In: os.Stdin},
		AppExecutable: getAppExecutable(),
		Exit:          os.Exit,
		UserCacheDir:  userCacheDir,
		DemoMode:      strings.ToLower(demoModeEnv) == "true",
	}
	commands.ExecuteCommand(appConfig, os.Args[1:])
}

func getAppExecutable() string {
	appExecutable, err := os.Executable()
	if err != nil {
		panic(fmt.Sprint("Cannot determine executable ", err))
	}
	// Escape back slashes so the exectuable works from GitBash on Windows
	appExecutable = strings.ReplaceAll(appExecutable, "\\", "\\\\")
	// Quote in case the path has a space.
	return "\"" + appExecutable + "\""
}
