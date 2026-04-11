# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

**IMPORTANT: Never call `go test` directly. Always use `make` with `TEST_ARGS` and include `-timeout 10s` to avoid hanging tests.**

```bash
# Build the project (formats, builds binary to ./bin/)
make build

# Run linter (requires golangci-lint installed)
make lint

# Run all tests (includes build and lint)
make test

# Run tests without linting
make test -o lint

# Run a specific test
make TEST_ARGS="-timeout 10s -run TestSdUpdate_WhenDestinationCommitNotSpecified" -o lint test

# Build and manually test a command
make build && ./bin/gh-stacked-diff <command-name>
```

## Code Architecture

### Project Purpose
This is a CLI tool (`sd`) that implements a stacked diff workflow for Git/GitHub. It allows developers to work with multiple small PRs stacked on their local main branch without constantly switching branches. The tool manages git commits, branches, and GitHub PRs through the GitHub CLI.

### Entry Point and Command Structure
- **Entry point**: `main.go` creates an `AppConfig` and calls `commands.ExecuteCommand()`
- **Command framework**: Uses `github.com/spf13/cobra` for CLI parsing, subcommands, help generation, and shell completions
- **Command registration**: `commands/parse_arguments.go` contains `buildRootCommand(appConfig)` which creates the root `*cobra.Command` and adds all subcommands via `rootCmd.AddCommand(...)`
- **Command pattern**: Each command is a `*cobra.Command` returned by a `create<CommandName>Command(appConfig util.AppConfig)` factory function
- **AppConfig injection**: `appConfig` is passed to each factory function and captured via closure in the command's `Run` function. No global state.

### Command Implementation Pattern
Commands follow this structure:
1. Each command has two files: `commands/command_<name>.go` and `commands/command_<name>_test.go`
2. Each command file exports a `create<CommandName>Command(appConfig util.AppConfig)` function that returns `*cobra.Command`
3. Common flag helpers: `addIndicatorFlag(cmd)`, `addReviewersFlags(cmd)` for reusable functionality
4. Commands use `getTargetCommits()` for interactive commit selection
5. **Annotations**: `DefaultLogLevel` and `CheckRepo` are stored in `cobra.Command.Annotations` map:
   - `"defaultLogLevel"`: `"error"` for output commands, omit for default `"info"`
   - `"checkRepo"`: `"true"` if command needs a git repo (omit annotation if command doesn't need a repo)
6. **Short flags**: Common flags use short forms (e.g. `-i`, `-r`, `-l`). Run `sd <command> --help` to see available flags and their short forms

### Key Modules

**commands/** - All CLI commands
- Uses cobra/pflag for argument parsing with `--long` and `-s` short flag support
- Commands are registered in `parse_arguments.go` alphabetically in `buildRootCommand()`
- `PersistentPreRun` on root command handles log level setup and repo checks
- Helper functions: `getTargetCommits()`
- Shell completions are provided by cobra's built-in `completion` command

**util/** - Core infrastructure
- `AppConfig`: Dependency injection struct for testing (holds IO, Exit function, paths)
- `execute.go`: Command execution with `ExecuteOrDie()` and `ExecuteOptions`
- `git_util.go`: Low-level git helpers (`GetCurrentBranchName`, `GetRepoName`) needed by infrastructure
- `TestExecutor`: Mock executor for unit tests (allows faking git/gh responses)

**gitutil/** - Git and GitHub operations
- `git_util.go`: Branch detection (local/remote main, worktrees), stashing, cherry-pick, rebase
- `gh_util.go`: GitHub CLI interactions (PR status, checks, reviews)
- `gh_pr_info.go`: PR info queries (merged/unmerged PR lookup)
- `gh_repo.go`: Repository metadata (name with owner, hostname)
- `gh_checks.go`: CI check status with historical min-checks caching
- `git_rollback.go`: Rollback manager for safe git operations

**templates/** - PR template system and git log parsing
- `GitLog` struct: Represents a commit with its branch and metadata
- `GetNewCommits()`: Parses commits between local main and origin/main
- Template rendering for PR titles, descriptions, and branch names
- Users can customize templates in `~/.gh-stacked-diff/`

**interactive/** - Terminal UI components
- Uses charmbracelet/bubbletea for interactive selection
- `CommitSelectionOptions`: Configures commit picker behavior
- `UserSelection()`: Interactive reviewer selection

**testutil/** - Testing infrastructure
- `InitTest()`: Sets up isolated git repo for each test in temp directory
- `AddCommit()`: Helper to create commits in tests
- `TestExecutor`: Mock for git/gh commands
- Tests run in `${UserCacheDir}/gh-stacked-diff-tests/<test-name>/`

### Commit Indicators
The tool supports three ways to reference commits (via the `--indicator`/`-i` flag):
- `commit`: Git commit hash (abbreviated or full)
- `pr`: GitHub PR number
- `list`: Index from `sd log` output (0-99)
- `guess` (default): Automatically determines type based on format

### Testing Strategy
- Each command has corresponding `*_test.go` file
- Tests use `testutil.InitTest()` to create isolated git environments with real git commands (not mocked) — these are integration tests, not unit tests
- Set log level to `slog.LevelDebug` in tests for detailed output
- Use `TestExecutor.SetResponse()` to fake only gh responses that can't run locally (e.g., `gh pr create`)
- Helper: `testParseArguments()` simulates command execution via `buildRootCommand()` + `rootCmd.Execute()`

## Creating New Commands

To add a new command:

1. Create `commands/command_<name>.go` with `create<CommandName>Command(appConfig util.AppConfig)` function returning `*cobra.Command`
2. Create `commands/command_<name>_test.go` with test cases
3. Register in `commands/parse_arguments.go` (add to `rootCmd.AddCommand(...)`, keep alphabetical)
4. Follow existing command patterns:
   - Use `addIndicatorFlag(cmd)` if command selects commits
   - Use `addReviewersFlags(cmd)` if command works with PRs
   - Use `getTargetCommits()` for interactive commit selection
   - Set `Annotations: map[string]string{"defaultLogLevel": "error"}` for output commands
   - Set `Annotations: map[string]string{"checkRepo": "true"}` if command needs a git repo (omit for commands that don't)
   - Use `Args: cobra.NoArgs`, `cobra.MaximumNArgs(1)`, `cobra.MinimumNArgs(2)`, etc. for argument validation

## Important Conventions

- **Main branch**: Tool supports either `main` or `master` via `gitutil.GetLocalMainBranchOrDie()` (respects worktrees) and `gitutil.GetRemoteMainBranchOrDie()`
- **Branch naming**: Auto-generated from commit subject (sanitized, max 120 chars)
- **Error handling**: Use `panic()` for user-facing errors (caught by defer/recover in `ExecuteCommand` and `testParseArgumentsWithOut`). This is an intentional design choice, not a code smell. The panic/recover pattern keeps command implementations clean — commands just `panic("message")` on errors and the top-level recover catches them, prints the message, and exits gracefully. Do NOT refactor this to use returned errors; that would add boilerplate to every command for no benefit
- **Logging**: Use `slog` (Debug, Info, Warn, Error levels). Always pass a single `fmt.Sprint(...)` message string — do NOT use slog's key-value args (e.g. `slog.Debug(fmt.Sprint("msg: ", val))` not `slog.Debug("msg", "key", val)`), as the key-value format produces undesirable output with the project's custom logger
- **IO abstraction**: Always use `appConfig.Io.Out/Err/In` instead of `os.Stdout/Stderr/Stdin`
- **Exit abstraction**: Use `appConfig.Exit()` instead of `os.Exit()` for testability
- **Command execution**: Use `util.ExecuteOrDie()` or `util.Execute()` (never call git/gh directly)

## Dependencies

- Go 1.24+
- `github.com/spf13/cobra` for CLI framework
- Uses forked versions of charmbracelet libraries (`github.com/joshallenit/bubbles` and `github.com/joshallenit/bubbletea`)
- Requires `gh` (GitHub CLI) and `git` to be installed and in PATH
- Requires `golangci-lint` for linting

## Development Notes

- Windows support: Uses Git Bash, handles path separators and escaping
- Platform-specific code: Check `runtime.GOOS` for Darwin/Windows differences
- Demo mode: Set `GH_STACKED_DIFF_DEMO_MODE=true` environment variable
