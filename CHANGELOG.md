# Change Log

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.1.0](https://github.com/slackhq/gh-stacked-diff/compare/v2.0.12...v2.1.0) - 2026-03-23

### Added

- `log` now has "--status" and "--poll" flags that displays Github PR status. 

### Changed

- `update` now waits 10 seconds before checking status.
- All poll intervals now use the config pollInterval value.
- `--merge` flag now implies to mark PR as ready when checks pass, so the prompt is skipped.

## [2.0.12](https://github.com/slackhq/gh-stacked-diff/compare/v2.0.11...v2.0.12) - 2026-03-22

### Added

- Shell auto-completions. See entry in README for installation instructions.
- One-character shortcuts for flags.
- promptForReview config flag that determines if a prompt to mark PR as ready for review after `new` and `update` commands is shown. Possible values are:
    - `never`: prompt is not shown
    - `promptY`: prompt with a default of Yes
    - `promptN`: prompt with a default of No

### Changed

- Migrated to Cobra library for options so now global options can be specified before or after command name. This also means that the long names of the flags now require two dashes "--log-level" vs. "-log-level".
- /dev/null is used for no-hooks directory instead of non-exisitent directory.
- `add-reviewers` command can now be used to only mark a PR as ready without adding reviewers.

### Fixed

- `prs` command shows inline now instead of using the full screen pager.


## [2.0.11](https://github.com/tinyspeck/gh-stacked-diff/compare/v2.0.10...v2.0.11) - 2026-03-12

### Fixed

- Unselecting a row sometimes highlighted top row.

## [2.0.10](https://github.com/tinyspeck/gh-stacked-diff/compare/v2.0.9...v2.0.10) - 2026-03-12

### Changed

- Up/down during commit selection skips disabled rows in the table.
- Changed default answer of prompt: "Mark PR as ready for review when checks pass?" to Yes.
- `replace-commit` command now rolls back on error. If the error happens during cherry-pick it asks the user if they want to rollback or continue the cherry-pick manually. There is a new flag `-on-cherry-pick-error` with three options `prompt`, `rollback`, or `exit`.
- `update` now can commits older than the PR commit to the PR.

## [2.0.9](https://github.com/tinyspeck/gh-stacked-diff/compare/v2.0.8...v2.0.9) - 2026-02-27

### Added

- Confirmation on `new` and `update`: "Mark PR as ready for review when checks pass?". This makes it simpler when you just want to mark a PR as ready without adding any reviewers. Often CODEOWNERS will automatically add reviewers.
- `--merge` flag on `new`, `update`, and `add-reviewers`. Enables auto-merge on the PR when checks pass.
- `new` now pushes last to avoid a situation where the fix-up on `main` fails, and the local branches are rolled back, but the push to origin already happened.

### Fixed

- Do not include commit comments (usually about conflicts) from body when creating PR description.

### Contributors

Special thanks to the following contributors for contributing to this release!

- [Ritesh Bhatia - @ritesh.bhatia](https://github.com/ritesh.bhatia)

## [2.0.8](https://github.com/tinyspeck/gh-stacked-diff/compare/v2.0.7...v2.0.8) - 2026-02-13

### Fixed

- Move no-hooks directory to user cache directory so that if it is created it is not in repository.

## [2.0.7](https://github.com/tinyspeck/gh-stacked-diff/compare/v2.0.6...v2.0.7) - 2026-02-12

### Fixed

- Do not run git hooks on commits that rebase is doing

## [2.0.6](https://github.com/tinyspeck/gh-stacked-diff/compare/v2.0.5...v2.0.6) - 2026-02-11

### Fixed

- Fixed get collaborators so it works on Github Enterprise

## [2.0.5](https://github.com/tinyspeck/gh-stacked-diff/compare/v2.0.4...v2.0.5) - 2026-02-09

### Changed

- add-reviewers include number of checks

## [2.0.4](https://github.com/tinyspeck/gh-stacked-diff/compare/v2.0.3...v2.0.4) - 2026-02-09

### Changed

- Do not run git hooks on rebase.
- Increase maximum number of checks for add-reviewers.

## [2.0.3](https://github.com/tinyspeck/gh-stacked-diff/compare/v2.0.2...v2.0.3) - 2026-01-28

### Added

- `migrate` command to prepares local git repository for first use by sd.

## [2.0.2](https://github.com/tinyspeck/gh-stacked-diff/compare/v2.0.1...v2.0.2) - 2025-12-04

### Changed

- `sd rebase-main` deletes branches for closed PRs.
- Better messaging when rebase fails.
- `sd log` and interactive prompts: shorter display of branches with many commits.

### Contributors

Special thanks to the following contributors for contributing to this release!

- [Hossain Khan - @hossain-khan](https://github.com/hossain-khan)
- [Ritesh Bhatia - @ritesh.bhatia](https://github.com/ritesh.bhatia)

## [2.0.1](https://github.com/tinyspeck/gh-stacked-diff/compare/v2.0.0...v2.0.1) - 2026-06-13

### Changed

- retry `gh` commands to account for network failures.

## [2.0.0](https://github.com/tinyspeck/gh-stacked-diff/compare/v1.3.0...v2.0.0) - 2025-06-09

### Added

- Interactive UI for selecting commits
- Interactive UI for selecting users to request review for
- Ability to use log list index for a commit indicator. Avoids having to copy & paste git hashes or PR numbers.
- Ability to add reviewers from `new` and `update` commands.
- `sd log` now also displays commits on associated branches.
- Ability to set log level via `sd` flag "--log-level".
- Unit tests.
- Build actions so that this can be used as a Github CLI plugin.

### Changed

- Moved all scripts to subcommands of a new `sd` executable.
- `sd rebase-main` now outputs in real time instead of only when rebase ends.
- `sd rebase-main` now deletes branches that have been merged.
- Converted all logs to use slog (logs at DEBUG, INFO, or ERROR levels) so that the log level can be changed to help with debugging.
- Renamed replace-head to replace-conflicts
- `sd log` was made faster by running some git commands once, instead of for each commit.
- `sd replace-conflicts` now asks for confirmation (also has a confirm flag).
- Reorganized code so that project can be used as a Github CLI plugin and go library.
- Removed scripts under /src/bash that were deprecated.
- Renamed repository from stacked-diff-workflow to gh-stacked-diff. The gh- prefix is required for it to work as a Github CLI plugin.
- More reliable rollback for "new" and "update" command when there is a problem.

### Fixed

- More reliable `getMainBranch`.
- More reliable help and command line parsing error messages.
- More reliable `sd rebase-main` by using Github CLI to check for merged branches.

## [1.3.3](https://github.com/tinyspeck/gh-stacked-diff/compare/v1.3.2...v1.3.3) - 2023-06-12

### Added

- Code owners: Now you can see which files caused code owners to be added as reviewers to a PR. Part of the default PR description template, these are added to the comment section of the PR description.

### Changes

- Improvements to install-apk so that it timesout, kills adb, and retries for when adb is stuck
- Separation of install-app and install-apk so that install-apk can be done without going through configuration
- Improvements to assemble-app: addition of assertMaxHeight

## [1.3.2](https://github.com/tinyspeck/gh-stacked-diff/compare/v1.3.0...v1.3.1) - 2023-02-01

### Added

- Support for repositories that use master as their main branch instead of main.

### Changes

- Continue executing new-pr if web browser could not open to PR for any reason.
- Automatically converts / to - in branch name to support webapp out of the box without having to change the branch name template.

## [1.3.1](https://github.com/tinyspeck/gh-stacked-diff/compare/v1.3.0...v1.3.1) - 2023-01-27

### Changes

- Simplified logs. update-pr and new-pr now have a log format flag (-logFlags) that can be used to include time.
- update-pr: rebases with remote branch when necessary
- update-pr: autostashes changes
- assemble-app includes other tasks that were not executed by plain detekt: detektRelease and detektReleaseUnitTest

## [1.3.0](https://github.com/tinyspeck/gh-stacked-diff/compare/v1.0...v1.3) - 2023-01-03

### Added

- Better usage documentation. Use -h flag in new-pr, update-pr, and add-reviewers to see
- New scripts to support custom scripting: wait-for-merge and get-branch-name-for
- Silent mode for builds that use say: Use silent flag (-s) for install-app, assemble-app
