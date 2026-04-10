# Developer Guide

## How To Build

The Stacked Diff Workflow CLI (also known as, the `sd` command) is
written in golang.

1. Install [golang](https://go.dev/dl/).
2. Install make. This is already installed on Mac. Instructions for
   windows are [here](https://leangaurav.medium.com/how-to-setup-install-gnu-make-on-windows-324480f1da69).
3. Install lint.
   [Install instructions](https://golangci-lint.run/welcome/install/#local-installation)

Then run:

```bash
make build
```

Binaries are created in `./bin`.

### Installation From Source

Clone the repository, [then build](DEVELOPER_GUIDE.md#how-to-build), and
then follow the [install instructions](#installation-from-a-release) for
your platform.

## Code Organization

The main entry point to the Stacked Diff Workflow CLI ("sd") is
[main.go]. The commands are implemented under [commands].

## How to Debug Unit Tests

`make test` will run the unit tests. To skip lint use
`make test -o lint`. Check out the `TEST_ARGS` example in `Makefile` to
run only some test.

If one of the command*_test fails you can use
`testutil.InitTest(t, slog.LevelDebug)` for more detailed logging. This
will cause `testParseArguments` to add a `"--log-level=debug` to the
command line arguments.

You can use TEST_ARGS to run only one test from the command line, see
[Makefile].

## Making a Release

Follow the steps in golang docs
[Publishing a module](https://go.dev/doc/modules/publishing):

```bash
make update-readme
# - Make addition to [CHANGELOG.md], including Contributors section.
# - Update the stable version so that it is equal to current version [util/stable_version.txt]
# Then...
# - create PR, merge changes, and then:
sd rebase-main
# - Make sure all changes merged into main, git status and sd log should
#   be empty. Otherwise save your changes, "git reset --hard origin/main",
#   create tag, then restore your changes
git status && sd log
# Then make the release:
export RELEASE_VERSION=`cat util/stable_version.txt`
git tag v$RELEASE_VERSION
git push origin v$RELEASE_VERSION
# - Increment [util/current_version.txt] so that the version lists as preview until next release.
```

For bubbles and bubbletea forks:

```bash
# Make labels in their repositories
git checkout main
git status && sd log
export RELEASE_VERSION=1.3.6
git tag v$RELEASE_VERSION
git push origin v$RELEASE_VERSION
# sd rebase-main if required
GOPROXY=proxy.golang.org go list -m \
  github.com/slackhq/bubbletea@v$RELEASE_VERSION
# In gh-stacked-diff:
# Update version in go.mod replace, then:
# Remove go.work and go.work.sum if using them:
mv go.work ../gh-stacked-diff-go.work
mv go.work.sum ../gh-stacked-diff-go.work.sum
go mod tidy

# Same steps for:
GOPROXY=proxy.golang.org go list -m \
  github.com/slackhq/bubbles@v$RELEASE_VERSION
```

Once a tag is created [.github/workflows/release.yml] kicks off and
creates the binaries for the release.
