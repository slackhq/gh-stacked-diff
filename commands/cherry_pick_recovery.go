package commands

import (
	"fmt"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/interactive"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

const (
	onCherryPickErrorPrompt   = "prompt"
	onCherryPickErrorRollback = "rollback"
	onCherryPickErrorExit     = "exit"
)

type cherryPickRecoveryOptions struct {
	// OnCherryPickError controls behavior when cherry-pick fails:
	// "prompt" (default), "rollback", or "exit".
	OnCherryPickError string
	// OnRollback is called before re-panicking when the user chooses to rollback.
	OnRollback func()
	// OnContinueManually is called before exiting when the user chooses to continue manually.
	OnContinueManually func()
	// AdditionalResolveSteps are printed after the standard resolve steps.
	AdditionalResolveSteps []string
	// AdditionalAbortSteps are printed after "git cherry-pick --abort".
	AdditionalAbortSteps []string
}

// cherryPickWithRecovery runs CherryPickAndSkipAllEmpty and handles failures
// by prompting the user to rollback or continue manually.
func cherryPickWithRecovery(gitDir string, commits []string, opts cherryPickRecoveryOptions) {
	cherryPickErr := func() (r any) {
		defer func() { r = recover() }()
		gitutil.CherryPickAndSkipAllEmpty(gitDir, commits)
		return nil
	}()
	if cherryPickErr == nil {
		return
	}
	appConfig := util.GetAppConfig()
	util.Fprintln(appConfig.Io.Out, fmt.Sprint("Cherry-pick failed: ", cherryPickErr))
	onError := opts.OnCherryPickError
	if onError == "" {
		onError = onCherryPickErrorPrompt
	}
	shouldRollback := onError == onCherryPickErrorRollback ||
		(onError == onCherryPickErrorPrompt && interactive.Confirm("Rollback all changes?", true))
	if shouldRollback {
		if opts.OnRollback != nil {
			opts.OnRollback()
		}
		panic(cherryPickErr)
	}
	util.Fprintln(appConfig.Io.Out, "To resolve manually:")
	util.Fprintln(appConfig.Io.Out, "  1. Fix the conflicts")
	util.Fprintln(appConfig.Io.Out, "  2. git add <resolved files>")
	util.Fprintln(appConfig.Io.Out, "  3. git cherry-pick --continue")
	util.Fprintln(appConfig.Io.Out, "  Repeat steps 1-3 until cherry-pick is complete.")
	for _, step := range opts.AdditionalResolveSteps {
		util.Fprintln(appConfig.Io.Out, step)
	}
	util.Fprintln(appConfig.Io.Out, "To abort:")
	util.Fprintln(appConfig.Io.Out, "  1. git cherry-pick --abort")
	for _, step := range opts.AdditionalAbortSteps {
		util.Fprintln(appConfig.Io.Out, step)
	}
	if opts.OnContinueManually != nil {
		opts.OnContinueManually()
	}
	appConfig.Exit(0)
}
