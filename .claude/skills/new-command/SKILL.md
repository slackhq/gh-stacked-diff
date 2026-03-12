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
	"flag"
	"log/slog"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func create<CommandName>Command() Command {
	flagSet := flag.NewFlagSet("<command-name>", flag.ContinueOnError)

	// Add command-specific flags here
	// Example: someFlag := flagSet.String("flag-name", "default", "Description")

	return Command{
		FlagSet:         flagSet,
		DefaultLogLevel: slog.Level<LogLevel>,
		Summary:         "<summary>",
		Description: `<Detailed description>

Examples:
  sd <command-name>
  sd <command-name> --flag value`,
		Usage:         "<usage>",
		SkipRepoCheck: <skipRepoCheck>,
		OnSelected: func(appConfig util.AppConfig, command Command) {
			// Parse and validate arguments
			if flagSet.NArg() > 0 {
				commandError(appConfig, flagSet, "unexpected arguments", command.Usage)
			}

			// TODO: Implement command logic here
			util.Fprintln(appConfig.Io.Out, "Command executed successfully")
		},
	}
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
   - Read `commands/parse_arguments.go` to find the commands slice (around line 66-82)
   - Add `create<CommandName>Command(),` to the list in alphabetical order
   - Mark this todo as completed

8. **Run Tests**: Mark this todo as in_progress
   - Run `make TEST_ARGS="-run TestSd<CommandName>" test` to verify the code compiles and basic tests pass
   - If tests fail, report the error to the user and keep this todo as in_progress
   - If tests pass, mark this todo as completed

9. **Provide Next Steps**: Mark this todo as in_progress. Tell the user:
   - The command and test files have been created
   - Basic tests have passed successfully
   - They need to implement the actual command logic in the OnSelected function
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
- `<skipRepoCheck>`: `true` or `false`
- `<LogLevel>`: `Error` or `Info`

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
- Commands that output structured data should use DefaultLogLevel: slog.LevelError
- Commands that show progress should use DefaultLogLevel: slog.LevelInfo
