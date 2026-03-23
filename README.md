<p align="center">
  <img
    src="docs/README-sd-preview.png"
    alt="Stacked Diff Workflow. Stay on main, skip the branches. Put up pull requests in slices.">
</p>

Using a [stacked diff workflow](https://newsletter.pragmaticengineer.com/p/stacked-diffs) and `sd` allows you to:

- Break down a pull request into several smaller PRs.
- Work on separate streams of work without the overhead of changing branches.
- Have local commits that are always present and are never pushed. For example, logging that helps you debug your changes but is too noisy for others.
- Quickly create and update pull requests.
- Add reviewers only once PR checks have passed.

Once you experience the efficiency of stacked diffs you can't imagine going back to your old workflow.

This project is a Command Line Interface (`sd`) that manages git commits and branches to allow you to quickly use a stacked diff workflow. It uses the Github CLI to interact with Github.

## Installation

### Installation as Github CLI Plugin

#### Mac

*Optional: As this is a CLI, do yourself a favor and install [iTerm](https://iterm2.com/) and [zsh](https://ohmyz.sh/), as they make working from the command line more pleasant.*

```bash
# Install Github CLI.
brew install gh
# Setup login for Github CLI
gh auth login
# Install plugin
gh extensions install slackhq/gh-stacked-diff
# Add a shell function to make it faster to use.
# For example if using zsh (note: must be a function and not an alias for shell completions to work):
echo 'sd() { gh stacked-diff "$@"; }' >> ~/.zshrc
# Enable shell completions for zsh:
echo 'eval "$(sd completion zsh)"' >> ~/.zshrc
source ~/.zshrc
```

#### Windows

1. Install [Git and Git Bash](https://gitforwindows.org/)
2. Install [Github CLI](https://cli.github.com/). Winget is possible: `winget install --id GitHub.cli`
3. Authenticate gh and install plugin:

      ```bash
      gh auth login
      # Install plugin
      gh extensions install slackhq/gh-stacked-diff
      # Add a shell function to make it faster to use.
      # For example if using Git Bash:
      echo 'sd() { gh stacked-diff "$@"; }' >> ~/.bashrc
      # Enable shell completions for bash:
      echo 'eval "$(sd completion bash)"' >> ~/.bashrc
      source ~/.bashrc
      ```

## Command Line Interface

```
Usage:
  sd [command]

Available Commands:
  add-reviewers       Add reviewers to Pull Request on Github once its checks have passed
  branch-name         Outputs branch name of commit
  checkout            Checks out branch associated with commit indicator
  code-owners         Outputs code owners for all of the changes in branch
  completion          Generate the autocompletion script for the specified shell
  help                Help about any command
  log                 Displays git log of your changes
  migrate             Migrates any work-in-progress branches to main. This prepares local git repository for first use by sd.
  new                 Create a new pull request from a commit on main
  prs                 Lists all Pull Requests you have open.
  rebase-main         Bring your main branch up to date with remote
  replace-commit      Replaces a commit on main branch with its associated branch
  replace-conflicts   For failed rebase: replace changes with its associated branch
  update              Add commits from main to an existing PR
  version             Outputs version number
  wait-for-merge      Waits for a pull request to be merged

Flags:
  -c, --config stringToString   Set a config value as key=value. Overrides values from
                                ~/.gh-stacked-diff/config.yaml. Supported keys:
                                   promptForReview=never|promptY|promptN (default: promptN)
                                   pollInterval=<duration> (default: 30s, e.g. 1m, 10s)
                                Can be specified multiple times for different keys.

                                Equivalent config.yaml:
                                   promptForReview: promptY
                                   pollInterval: 1m (default [])
  -h, --help                    help for sd
  -l, --log-level string        Possible log levels:
                                   debug
                                   info
                                   warn
                                   error
                                Default is info, except on commands that are for output purposes,
                                (namely branch-name and log), which have a default of error.
  -v, --version                 version for sd

Use "sd [command] --help" for more information about a command.
```

### Basic Commands

#### log

Displays summary of the git commits on current branch that are not in the remote branch.

Useful to view list indexes, or copy commit hashes, to use for the commitIndicator required by other commands.

A ✅ means that there is a PR associated with the commit (actually it means there is a branch, but having a branch means there is a PR when using this workflow). If there is more than one commit on the associated branch, those commits are also listed (indented under their associated commit summary).

```
usage: sd log [flags]

Flags:
  -s, --status   Show PR status including checks, approvals, and state.
                 Only supported on the main branch.
  -p, --poll     Keep polling for status updates. Implies --status.
                 Press Esc or Ctrl+C to exit.
                 Poll interval is configurable via --config pollInterval=30s
```

<img width="663" alt="image" src="docs/README-sd-log.png">

#### new

Create a new PR with a cherry-pick of the given commit indicator.

This command first creates an associated branch, (with a name based on the commit summary), and then uses Github CLI to create a PR.

Can also add reviewers once PR checks have passed, see "--reviewers" flag.

<img width="663" alt="image" src="docs/README-sd-new.gif">

```
usage: sd new [commitIndicator] [flags]

If commitIndicator is missing then you will be prompted to select commit:

   [enter]    confirms selection
   [up,k]     moves cursor up
   [down,j]   moves cursor down
   [q,esc]    cancels

Ticket Number:

If you prefix a (Jira-like formatted) ticket number to the git commit
summary then the "Ticket" section of the PR description will be
populated with it.

For example:

"CONV-9999 Add new feature"

Templates:

The Pull Request Title, Body (aka Description), and Branch Name are
created from golang templates.

The default templates are:

   branch-name.template:      templates/config/branch-name.template
   pr-description.template:   templates/config/pr-description.template
   pr-title.template:         templates/config/pr-title.template

To change a template, copy the default from templates/config/ into
~/.gh-stacked-diff/ and modify contents.

The possible values for the templates are:

   CommitBody                   Body of the commit message
   CommitSummary                Summary line of the commit message
   CommitSummaryCleaned         Summary line of the commit message without
                                spaces or special characters
   CommitSummaryWithoutTicket   Summary line of the commit message without
                                the prefix of the ticket number
   FeatureFlag                  Value passed to feature-flag flag
   TicketNumber                 Jira ticket as parsed from the commit summary
   Username                     Name as parsed from git config email.
   UsernameCleaned              Username with dots (.) converted to dashes (-).

flags:

  -b, --base string           Base branch for Pull Request. Default is main
  -d, --draft                 Whether to create the PR as draft (default true)
  -f, --feature-flag string   Value for FEATURE_FLAG in PR description
  -i, --indicator string      Indicator type to use to interpret commitIndicator:
                                 commit   a commit hash, can be abbreviated,
                                 pr       a github Pull Request number,
                                 list     the order of commit listed in the git log, as indicated
                                          by "sd log"
                                 guess    the command will guess the indicator type:
                                    Number between 0 and 99:       list
                                    Number between 100 and 999999: pr
                                    Otherwise:                     commit
                               (default "guess")
  -m, --merge                 Enable auto-merge (squash) on the PR via Github CLI.
                              Implies marking the PR as ready for review.
      --min-checks int        Minimum number of checks to wait for before verifying that checks
                              have passed before adding reviewers. It takes some time for checks
                              to be added to a PR by Github, and if you add-reviewers too soon it
                              will think that they have all passed. Default of -1 means to use 4
                              or the average number of checks of merged PRs, whatever is less. (default -1)
  -r, --reviewers string      Comma-separated list of Github usernames to add as reviewers once
                              checks have passed.
  -s, --silent                Whether to use voice output (false) or be silent (true) to notify that reviewers have been added.
```

###### Note on Commit Messages

Keep your commit summary to a [reasonable length](https://www.midori-global.com/blog/2018/04/02/git-50-72-rule). The commit summary is used as the branch name. To add more detail, use the [commit description](https://stackoverflow.com/questions/40505643/how-to-do-a-git-commit-with-a-subject-line-and-message-body/40506149#40506149). The
created branch name is truncated to 120 chars as Github has problems with very long
branch names.

#### update

Add commits from local main branch to an existing PR.

Can also add reviewers once PR checks have passed, see "--reviewers" flag.

<img width="663" alt="image" src="docs/README-sd-update.gif">

```
usage: sd update [PR commitIndicator [fixup commitIndicator...]] [flags]

If commitIndicators are missing then you will be prompted to select commits:

   [enter]    confirms selection
   [space]    adds to selection when selecting commits to add
   [up,k]     moves cursor up
   [down,j]   moves cursor down
   [q,esc]    cancels

flags:

  -i, --indicator string   Indicator type to use to interpret commitIndicator:
                              commit   a commit hash, can be abbreviated,
                              pr       a github Pull Request number,
                              list     the order of commit listed in the git log, as indicated
                                       by "sd log"
                              guess    the command will guess the indicator type:
                                 Number between 0 and 99:       list
                                 Number between 100 and 999999: pr
                                 Otherwise:                     commit
                            (default "guess")
  -m, --merge              Enable auto-merge (squash) on the PR via Github CLI.
                           Implies marking the PR as ready for review.
      --min-checks int     Minimum number of checks to wait for before verifying that checks
                           have passed before adding reviewers. It takes some time for checks
                           to be added to a PR by Github, and if you add-reviewers too soon it
                           will think that they have all passed. Default of -1 means to use 4
                           or the average number of checks of merged PRs, whatever is less. (default -1)
  -r, --reviewers string   Comma-separated list of Github usernames to add as reviewers once
                           checks have passed.
  -s, --silent             Whether to use voice output (false) or be silent (true) to notify that reviewers have been added.
```

#### add-reviewers

Add reviewers to Pull Request on Github once its checks have passed.

If PR is marked as a Draft, it is first marked as "Ready for Review".

```
usage: sd add-reviewers [commitIndicator...] [flags]

flags:

  -i, --indicator string          Indicator type to use to interpret commitIndicator:
                                     commit   a commit hash, can be abbreviated,
                                     pr       a github Pull Request number,
                                     list     the order of commit listed in the git log, as indicated
                                              by "sd log"
                                     guess    the command will guess the indicator type:
                                        Number between 0 and 99:       list
                                        Number between 100 and 999999: pr
                                        Otherwise:                     commit
                                   (default "guess")
  -m, --merge                     Enable auto-merge (squash) on the PR via Github CLI.
                                  Implies marking the PR as ready for review.
      --min-checks int            Minimum number of checks to wait for before verifying that checks
                                  have passed before adding reviewers. It takes some time for checks
                                  to be added to a PR by Github, and if you add-reviewers too soon it
                                  will think that they have all passed. Default of -1 means to use 4
                                  or the average number of checks of merged PRs, whatever is less. (default -1)
  -r, --reviewers string          Comma-separated list of Github usernames to add as reviewers once
                                  checks have passed.
  -s, --silent                    Whether to use voice output (false) or be silent (true) to notify that reviewers have been added.
  -w, --when-checks-pass          Poll until all checks pass before adding reviewers (default true)
```

### Commands for Rebasing and Fixing Merge Conflicts

#### rebase-main

Rebase with origin/main, dropping any commits whose associated branches have been merged or closed.

Commits from merged PRs are automatically dropped. For commits from closed (not merged) PRs, you will be prompted to confirm before dropping them.

This avoids having to manually call "git reset --hard head" whenever you have merge conflicts with a commit that has already been merged but has slight variation with local main because, for example, a change was made with the Github Web UI.

```
usage: sd rebase-main
```

#### checkout

Checks out the branch associated with commit indicator.

For when you want to merge only the branch with origin/main, rather than your entire local main branch, verify why CI is failing on that particular branch, or for any other reason.

After modifying the branch you can use "sd replace-commit" to sync local main.

```
usage: sd checkout [commitIndicator] [flags]

flags:

  -i, --indicator string   Indicator type to use to interpret commitIndicator:
                              commit   a commit hash, can be abbreviated,
                              pr       a github Pull Request number,
                              list     the order of commit listed in the git log, as indicated
                                       by "sd log"
                              guess    the command will guess the indicator type:
                                 Number between 0 and 99:       list
                                 Number between 100 and 999999: pr
                                 Otherwise:                     commit
                            (default "guess")
```

#### replace-commit

Replaces a commit on main branch with the squashed contents of its associated branch.

This is useful when you make changes within a branch, for example to fix a problem found on CI, and want to bring the changes over to your local main branch.

```
usage: sd replace-commit [commitIndicator] [flags]

flags:

  -i, --indicator string              Indicator type to use to interpret commitIndicator:
                                         commit   a commit hash, can be abbreviated,
                                         pr       a github Pull Request number,
                                         list     the order of commit listed in the git log, as indicated
                                                  by "sd log"
                                         guess    the command will guess the indicator type:
                                            Number between 0 and 99:       list
                                            Number between 100 and 999999: pr
                                            Otherwise:                     commit
                                       (default "guess")
      --on-cherry-pick-error string   Action when cherry-pick fails: prompt, rollback, or exit (default "prompt")
```

#### replace-conflicts

During a rebase that failed because of merge conflicts, replace the current uncommitted changes (merge conflicts), with the contents (diff between origin/main and HEAD) of its associated branch.

```
usage: sd replace-conflicts [flags]

flags:

  -y, --confirm   Whether to automatically confirm to do this rather than ask for y/n input
```

### Commands for Custom Scripting

#### branch-name

Outputs the branch name for a given commit indicator. Useful for your own custom scripting.

```
usage: sd branch-name [commitIndicator] [flags]

flags:

  -i, --indicator string   Indicator type to use to interpret commitIndicator:
                              commit   a commit hash, can be abbreviated,
                              pr       a github Pull Request number,
                              list     the order of commit listed in the git log, as indicated
                                       by "sd log"
                              guess    the command will guess the indicator type:
                                 Number between 0 and 99:       list
                                 Number between 100 and 999999: pr
                                 Otherwise:                     commit
                            (default "guess")
```

#### wait-for-merge

Waits for a pull request to be merged. Poll interval is configurable via `--config pollInterval`.

Useful for your own custom scripting.

```
usage: sd wait-for-merge [commitIndicator] [flags]

flags:

  -i, --indicator string   Indicator type to use to interpret commitIndicator:
                              commit   a commit hash, can be abbreviated,
                              pr       a github Pull Request number,
                              list     the order of commit listed in the git log, as indicated
                                       by "sd log"
                              guess    the command will guess the indicator type:
                                 Number between 0 and 99:       list
                                 Number between 100 and 999999: pr
                                 Otherwise:                     commit
                            (default "guess")
  -s, --silent             Whether to use voice output (false) or be silent (true) to notify that the PR has been merged.
```

### Other Commands

#### code-owners

Outputs code owners for each file that has been modified in the current local branch when compared to the remote main branch

```
usage: sd code-owners
```

#### prs

Lists all Pull Requests you have open.

You must be logged-in, via "gh auth login"

```
usage: sd prs
```

#### migrate

Migrates work-in-progress branches to main, preparing your local repository for stacked diff workflow.

This command is useful when first adopting sd in an existing repository with feature branches. It will help you move commits from feature branches onto your main branch so they can be managed as a stack.

```
usage: sd migrate
```

#### completion

Generate the autocompletion script for the specified shell. Supports bash, zsh, fish, and powershell.

See the [Installation](#installation-as-github-cli-plugin) section for how to enable shell completions.

```
usage: sd completion [bash|zsh|fish|powershell]
```

#### version

Outputs the version number.

```
usage: sd version
```

### Global Flags

The following flags are available on all commands:

```
  -c, --config stringToString   Set a config value as key=value. Overrides values from
                                ~/.gh-stacked-diff/config.yaml. Supported keys:
                                   promptForReview=never|promptY|promptN (default: promptN)
                                   pollInterval=<duration> (default: 30s, e.g. 1m, 10s)
                                Can be specified multiple times for different keys.

                                Equivalent config.yaml:
                                   promptForReview: promptY
                                   pollInterval: 1m (default [])
  -l, --log-level string        Possible log levels:
                                   debug
                                   info
                                   warn
                                   error
                                Default is info, except on commands that are for output purposes,
                                (namely branch-name and log), which have a default of error.
```

## Example Workflow

### Creating and Updating PRs

Use **sd new** and **sd update** to create and update PR's while always staying on `main` branch.

### To Update Main

*Note: This process is automated by the `sd rebase-main` command. There is no need to follow these steps manually.*

Once a PR has been merged, just rebase main normally. The local PR commit will be replaced by the one that Github created when squashing and merging.

```bash
git fetch && git rebase origin/main
```

If you run into conflicts with a commit that has already been merged you can just ignore it. This can happen, for example, if a change was made on github.com and it is not reflected in your local commit. Obviously, only do this if the PR has actually already been merged into main! The error message from rebase will let you know which commit has conflicts.

```bash
git reset --hard head && git rebase --continue
```

#### To Fix Merge Conflicts

##### Easy Flow

If you just are rebasing with `main` and the commit with merge conflict has already been **merged**, then the process is simpler.

1. Fix Merge Conflict

      ```bash
      # switch to feature branch that has a merge conflict
      sd checkout <commitIndicator>
      git fetch && git merge origin/main
      # ... and address any merge conflicts
      # Update your PR
      git push origin xxx
      ```

2. Merge PR via Github
3. [Update your Main Branch](#to-update-main)

##### Advanced Flow

If you want to update your main branch *before* you merge your PR, you can use **replace-conflicts** to keep your local `main` up to date.

```bash
# switch to feature branch that has a merge conflict
sd checkout <commitIndicator>
# rebase or merge
git fetch && git merge origin/main
# ... and address any merge conflicts
# Update your PR
git push origin xxx
# Rebase your local main branch.
git switch main
git rebase origin/main
# hit same merge conflicts, use replace-conflicts to copy the fixes you just made
replace-conflicts <commitIndicator>
# continue with the rebase
git add . && git rebase --continue
# All done... now both the feature branch and your local main are rebased with main,
# and the merge conflicts only had to be fixed once
```

## Building Source and Contributing

See the [Developer Guide](DEVELOPER_GUIDE.md), which includes instructions on how to build the source, as well as an overview of the code.

## Stacked Pull Requests?

Note: these scripts do *not* facilitate Stacked *Pull Requests*. Github does some things that add friction to using Stacked PR's, even with support from third party software. For example, after merging one of the PR's in the stack, the other PR's will require a re-review. Instead of Stacked PRs, it's recommended to organize your PR's, as much as reasonably possible, so that they can all be rebased against main at the same time. When there are dependencies, wait for dependant PR to be merged before putting up the next one. You may find that often you are still working on the next commit while the other is being reviewed/merged.

## Acknowledgments

- Thanks to [Dave Lee](https://x.com/kastiglione) for publishing [this article](https://kastiglione.github.io/git/2020/09/11/git-stacked-commits.html) that inspired the first version of the scripts.

- Thanks to the Github team for creating [their CLI](https://cli.github.com/) that is leveraged here.

## Version Compatibility

| Stacked Diff version | gh CLI versions tested | git versions tested |
| -------------------- | ---------------------- | ------------------- |
| [2.0.0](CHANGELOG.md#200---2025-02-28) | 2.38.0, 2.64.0, 2.66.1, 2.86.0 | 2.38.1, 2.47.1, 2.48.1, 2.51.1 |

## Troubleshooting

### Can push but not create a Pull Request

If you were added as a contributor to a project after you have already been logged in to `gh`, you will need to refresh your credentials.

You know you have push access, but `sd` fails with a message like this:
```
gh: Must have push access to view repository collaborators. (HTTP 403)
```

To fix:
```bash
gh auth refresh
```

Note that this only refreshes the **active** account, so if you are logged into more than one account on `github.com` you will have to `gh auth switch` to make sure the right account is active.
