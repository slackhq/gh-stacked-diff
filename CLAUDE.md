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
- **Command registration**: `commands/parse_arguments.go` contains the command registry (lines 66-82) where all commands are registered in a slice
- **Command pattern**: Each command is defined by a `Command` struct (in `commands/command.go`) with:
  - `FlagSet`: Command-specific flags
  - `OnSelected`: Function executed when command runs
  - `DefaultLogLevel`: Either `slog.LevelInfo` (for progress commands) or `slog.LevelError` (for output commands)
  - `Summary`, `Description`, `Usage`: Help text
  - `SkipRepoCheck`: Whether command requires a git repository

### Command Implementation Pattern
Commands follow this structure:
1. Each command has two files: `commands/command_<name>.go` and `commands/command_<name>_test.go`
2. Each command file exports a `create<CommandName>Command()` function that returns a `Command` struct
3. Common flag helpers: `addIndicatorFlag()`, `addReviewersFlags()` for reusable functionality
4. Commands use `getTargetCommits()` for interactive commit selection

### Key Modules

**commands/** - All CLI commands
- Uses `flag.FlagSet` for argument parsing with `flag.ContinueOnError`
- Commands are registered in `parse_arguments.go` alphabetically
- Helper functions: `commandError()`, `commandHelp()`, `getTargetCommits()`

**util/** - Shared utilities
- `AppConfig`: Dependency injection struct for testing (holds IO, Exit function, paths)
- `execute.go`: Command execution with `ExecuteOrDie()` and `ExecuteOptions`
- `git_util.go`: Git operations (branch detection, stashing, etc.)
- `gh_util.go`: GitHub CLI interactions
- `TestExecutor`: Mock executor for unit tests (allows faking git/gh responses)

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
The tool supports three ways to reference commits (via the `-indicator` flag):
- `commit`: Git commit hash (abbreviated or full)
- `pr`: GitHub PR number
- `list`: Index from `sd log` output (0-99)
- `guess` (default): Automatically determines type based on format

### Testing Strategy
- Each command has corresponding `*_test.go` file
- Tests use `testutil.InitTest()` to create isolated git environments
- Set log level to `slog.LevelDebug` in tests for detailed output
- Use `TestExecutor.SetResponse()` to fake git/gh CLI responses
- Helper: `testParseArguments()` simulates command execution

## Creating New Commands

To add a new command:

1. Create `commands/command_<name>.go` with `create<CommandName>Command()` function
2. Create `commands/command_<name>_test.go` with test cases
3. Register in `commands/parse_arguments.go` (add to commands slice, keep alphabetical)
4. Follow existing command patterns:
   - Use `addIndicatorFlag()` if command selects commits
   - Use `addReviewersFlags()` if command works with PRs
   - Use `getTargetCommits()` for interactive commit selection
   - Set `DefaultLogLevel` to `slog.LevelError` for output commands, `slog.LevelInfo` otherwise
   - Set `SkipRepoCheck: true` if command doesn't need a git repo

## Important Conventions

- **Main branch**: Tool supports either `main` or `master` via `util.GetMainBranchOrDie()`
- **Branch naming**: Auto-generated from commit subject (sanitized, max 120 chars)
- **Error handling**: Use `panic()` for user-facing errors (caught by `parseArguments` defer/recover). This is an intentional design choice, not a code smell. The panic/recover pattern keeps command implementations clean — commands just `panic("message")` on errors and the top-level recover in `parseArguments` catches them, prints the message, and exits gracefully. Do NOT refactor this to use returned errors; that would add boilerplate to every command for no benefit
- **Logging**: Use `slog` (Debug, Info, Warn, Error levels). Always pass a single `fmt.Sprint(...)` message string — do NOT use slog's key-value args (e.g. `slog.Debug(fmt.Sprint("msg: ", val))` not `slog.Debug("msg", "key", val)`), as the key-value format produces undesirable output with the project's custom logger
- **IO abstraction**: Always use `appConfig.Io.Out/Err/In` instead of `os.Stdout/Stderr/Stdin`
- **Exit abstraction**: Use `appConfig.Exit()` instead of `os.Exit()` for testability
- **Command execution**: Use `util.ExecuteOrDie()` or `util.Execute()` (never call git/gh directly)

## Dependencies

- Go 1.24+
- Uses forked versions of charmbracelet libraries (`github.com/joshallenit/bubbles` and `github.com/joshallenit/bubbletea`)
- Requires `gh` (GitHub CLI) and `git` to be installed and in PATH
- Requires `golangci-lint` for linting

## Development Notes

- Windows support: Uses Git Bash, handles path separators and escaping
- Platform-specific code: Check `runtime.GOOS` for Darwin/Windows differences
- Demo mode: Set `GH_STACKED_DIFF_DEMO_MODE=true` environment variable
- The tool can be used as a Go library (import `github.com/slackhq/gh-stacked-diff/v2`)
