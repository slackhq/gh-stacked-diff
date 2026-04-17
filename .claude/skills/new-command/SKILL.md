---
name: new-command
description: Creates a new command for gh-stacked-diff with associated test file and registers it in the command parser. Use this when the user wants to add a new CLI command to the project.
---

# new-command

Creates a new command for gh-stacked-diff with associated test file.

## Usage

```
/new-command [command-name]
```

## Instructions

When the user invokes this skill:

1. **Create Todo List**: Immediately use TodoWrite to create a todo list with these items:
   - Get command name from user (if not provided)
   - Validate command doesn't already exist
   - Gather command details from user
   - Create command file (commands/command_<name>.go)
   - Create test file (commands/command_<name>_test.go)
   - Register command in parse_arguments.go
   - Run tests to verify code compiles
   - Provide next steps to user

2. **Get Command Name**: Mark the first todo as in_progress
   - If provided as an argument, use it
   - If not provided, ask the user for the command name (use kebab-case like "branch-name", "wait-for-merge", etc.)
   - Once you have the name, mark this todo as completed

3. **Validate Command Name**: Mark this todo as in_progress
   - Check if `commands/command_<name>.go` already exists
   - If it exists, warn the user and ask if they want to continue
   - Mark this todo as completed

4. **Gather Command Details**: Mark this todo as in_progress
   - Ask the user using AskUserQuestion:
     * Summary: One-line description of what the command does
     * Usage: How to invoke the command (e.g., "sd <command-name> [flags] <args>")
     * Does the command need a git repository? (default: yes)
     * Default log level: info or error (use error for output commands, info for others)
     * Does the command need the indicator flag? (for selecting commits)
     * Does the command need reviewer flags? (for PR operations)
   - Mark this todo as completed

5. **Create Command File**: Mark this todo as in_progress. Create `commands/command_<name>.go` using this template:

```go
package commands

import (
	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/spf13/cobra"
)

func create<CommandName>Command(appConfig util.AppConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "<command-name>",
		Short: "<summary>",
		Long: `<Detailed description>

Examples:
  sd <command-name>
  sd <command-name> --flag value`,
		Args: cobra.NoArgs,
		// Add Annotations if needed:
		// Annotations: map[string]string{
		//   "defaultLogLevel": "error",    // for output commands (default is info)
		//   "checkRepo":       "true",     // if command needs a git repo (omit if not needed)
		// },
	}

	// Add command-specific flags here
	// Example: someFlag := cmd.Flags().StringP("flag-name", "f", "default", "Description")

	// Use addIndicatorFlag(cmd) if command selects commits
	// Use addReviewersFlags(cmd) if command works with PRs

	cmd.Run = func(cmd *cobra.Command, args []string) {
		// TODO: Implement command logic here
		util.Fprintln(appConfig.Io.Out, "Command executed successfully")
	}
	return cmd
}
```
   - Mark this todo as completed

6. **Create Test File**: Mark this todo as in_progress. Create `commands/command_<name>_test.go` using this template:

```go
package commands

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/slackhq/gh-stacked-diff/v2/testutil"
)

func TestSd<CommandName>_BasicTest(t *testing.T) {
	assert := assert.New(t)
	testutil.InitTest(t, slog.LevelError)

	// Add test setup (commits, branches, etc.)
	testutil.AddCommit("first commit", "file1.txt")

	// Execute the command
	out := testParseArguments("<command-name>")

	// Assert expected behavior
	assert.Contains(out, "expected output")
}

// TODO: Add more test cases
// - Error handling tests
// - Flag validation tests
// - Interactive selection tests (if applicable)
// - Edge cases
```
   - Mark this todo as completed

7. **Register Command**: Mark this todo as in_progress
   - Read `commands/parse_arguments.go` to find the `rootCmd.AddCommand(...)` block
   - Add `create<CommandName>Command(appConfig),` to the list in alphabetical order
   - Mark this todo as completed

8. **Run Tests**: Mark this todo as in_progress
   - Run `make TEST_ARGS="-timeout 10s -run TestSd<CommandName>" -o lint test` to verify the code compiles and basic tests pass
   - If tests fail, report the error to the user and keep this todo as in_progress
   - If tests pass, mark this todo as completed

9. **Provide Next Steps**: Mark this todo as in_progress. Tell the user:
   - The command and test files have been created
   - Basic tests have passed successfully
   - They need to implement the actual command logic in the Run function
   - They should add more test cases
   - They can run all tests with: `make test`
   - They can test the command manually with: `make build && ./bin/gh-stacked-diff <command-name>`
   - Mark this todo as completed

## Template Variables

Replace these placeholders when generating files:
- `<name>`: Command name in kebab-case (e.g., "branch-name")
- `<CommandName>`: Command name in PascalCase (e.g., "BranchName")
- `<command-name>`: Command name in kebab-case (for CLI usage)
- `<summary>`: One-line summary
- `<usage>`: Usage string

## Key Patterns

- Each `create*Command()` accepts `appConfig util.AppConfig` and returns `*cobra.Command`
- `appConfig` is captured via closure in `cmd.Run`
- Per-command metadata uses `cobra.Command.Annotations`:
  - `"defaultLogLevel": "error"` for output commands (default is info if not set)
  - `"checkRepo": "true"` for commands that need a git repo (omit if not needed)
- Error handling uses `panic("message")` — caught by defer/recover in `ExecuteCommand`
- Use `addIndicatorFlag(cmd)` for commit selection, `addReviewersFlags(cmd)` for PR operations
- Use `getTargetCommits(appConfig, args, indicatorTypeString, options)` for interactive commit selection

## Examples

```
/new-command list-commits
/new-command
```

## Notes

- Always use kebab-case for command names to match the existing convention
- Follow the patterns in existing commands for consistency
- Use the common flag helpers when appropriate (addIndicatorFlag, addReviewersFlags)
- Use getTargetCommits() for commands that need to select commits
- Commands that output structured data should use Annotations defaultLogLevel "error"
- Commands that show progress should use the default log level (info)
