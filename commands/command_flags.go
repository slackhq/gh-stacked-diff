package commands

import (
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func createFlagsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "flags",
		Short: "Details on global flags available to all commands",
		Long: `Global flags available to all commands:

--config, -c stringToString
  Set a config value as key=value. Overrides values from
  ~/.gh-stacked-diff/config.yaml. Can be specified multiple times.

  Supported keys:
    promptForReview=never|promptY|promptN (default: promptN)
    pollInterval=<duration> (default: 30s, e.g. 1m, 10s)
    ticketUrlPattern=<url> URL pattern for tickets, e.g.
                           ` + util.ExampleTicketUrlPattern + `
    worktreeMainBranchGuard=path|none (default: path)
       What to consider the "main" branch when in a worktree, to guard
       against incorrect use:
          path: worktree directory name
          none: current branch
    showWorktrees=true|false (default: true)
       Whether to show worktrees in log command
    showUiLegend=true|false (default: true)
       Whether to show keyboard shortcut legend in interactive UIs
    noTemplate=true|false (default: false)
       Use the commit body as the PR description without applying
       the PR description template

  Equivalent config.yaml:
    promptForReview: promptY
    pollInterval: 1m
    ticketUrlPattern: ` + util.ExampleTicketUrlPattern + `
    worktreeMainBranchGuard: path
    showWorktrees: true
    showUiLegend: true
    noTemplate: false

--log-level, -l string
  Possible log levels:
    debug
    info
    warn
    error
  Default is info, except on commands that are for output purposes,
  (namely branch-name and log), which have a default of error.`,
	}
}
