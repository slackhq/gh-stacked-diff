#!/bin/bash
# update-readme-help.sh
#
# Updates the generated portions of README.md to match the current binary:
#   1. The command summary table under "# Command Line Interface" is fully
#      regenerated from "sd --help".
#   2. The --help code block in each command section is refreshed. The Global
#      Flags in each block are replaced with a short summary that references the
#      full Global Flags section at the bottom of the README.
#   3. The Global Flags section at the bottom is refreshed.
#   4. A section is added (in alphabetical order) for any command that is
#      missing one.
#
# Usage: ./update-readme-help.sh

set -euo pipefail

cd "$(dirname "$0")"

BINARY="./bin/gh-stacked-diff"
README="README.md"
CONTENTFILE=$(mktemp)

# Commands that exist in the binary but intentionally have no README section
# or table row.
EXCLUDE="help version"

make build

TMPFILE=$(mktemp)
INSERTFILE=$(mktemp)
trap 'rm -f "$TMPFILE" "$INSERTFILE" "$CONTENTFILE"' EXIT

# get_command_names
# Prints the names of the binary's subcommands (one per line), excluding those
# in EXCLUDE.
get_command_names() {
    "$BINARY" --help 2>&1 | awk '
        /^Available Commands:$/ { in_cmds=1; next }
        in_cmds && /^$/         { exit }
        in_cmds                 { print $1 }
    ' | while IFS= read -r name; do
        case " $EXCLUDE " in
            *" $name "*) continue ;;
        esac
        printf '%s\n' "$name"
    done
}

# get_command_desc CMD
# Prints the one-line description of CMD from "sd --help".
get_command_desc() {
    local cmd="$1"
    "$BINARY" --help 2>&1 | awk -v c="$cmd" '
        /^Available Commands:$/ { in_cmds=1; next }
        in_cmds && /^$/         { exit }
        in_cmds && $1 == c      { $1=""; sub(/^ +/, ""); print; exit }
    '
}

# Extract the full global flags content for the bottom section from "sd --help".
# This captures the -c/--config and -l/--log-level blocks, skipping -h and -v.
GLOBAL_FLAGS_CONTENT=$("$BINARY" --help 2>&1 | awk '
    /^Flags:$/ { in_flags=1; next }
    !in_flags  { next }
    /^Use "/   { exit }
    /^$/       { next }
    /^  -c,/   { skip=0 }
    /^  -h,/   { skip=1 }
    /^  -l,/   { skip=0 }
    /^  -v,/   { skip=1 }
    !skip      { print }
')

# get_summarized_help CMD
# Runs CMD --help and replaces the Global Flags section with a 2-line summary.
get_summarized_help() {
    local cmd="$1"
    local help_output
    # shellcheck disable=SC2086
    help_output=$("$BINARY" $cmd --help 2>&1)

    # Print everything before "Global Flags:"
    echo "$help_output" | awk '/^Global Flags:$/ { exit } { print }'

    # Append summarized Global Flags
    printf 'Global Flags:\n'
    printf '  -c, --config stringToString   Set a config value as key=value (see Global Flags)\n'
    printf '  -l, --log-level string        Log level: debug, info, warn, error\n'
}

# generate_section CMD
# Prints a full README section for CMD: heading, description, and a collapsible
# --help code block (with summarized Global Flags). No trailing blank line.
generate_section() {
    local cmd="$1"
    printf '### %s\n\n' "$cmd"
    printf '%s\n\n' "$(get_command_desc "$cmd")"
    printf '<details>\n'
    printf '<summary><code>sd %s --help</code></summary>\n\n' "$cmd"
    printf '```\n'
    printf '%s\n' "$(get_summarized_help "$cmd")"
    printf '```\n\n'
    printf '</details>'
}

# build_table
# Prints the full "# Command Line Interface" summary table, regenerated from
# "sd --help".
build_table() {
    printf '| Command | Description |\n'
    printf '| --- | --- |\n'
    local name
    while IFS= read -r name; do
        printf '| [`%s`](#%s) | %s |\n' "$name" "$name" "$(get_command_desc "$name")"
    done < <(get_command_names)
}

# insert_section CMD SRC
# Reads README content from SRC and prints it with a new section for CMD
# inserted in alphabetical order (relative to the existing command sections),
# falling back to just before "## Global Flags".
insert_section() {
    local cmd="$1" src="$2" cmdset
    # Space-delimited set of known command names, used so we only compare
    # against "### " headings that are actually commands.
    cmdset=" $(get_command_names | tr '\n' ' ')"
    # Write the section to a file so awk can slurp it via getline (BSD awk does
    # not allow newlines in -v values).
    generate_section "$cmd" > "$CONTENTFILE"

    LC_ALL=C awk -v newcmd="$cmd" -v cmdset="$cmdset" -v contentfile="$CONTENTFILE" '
        function print_section(   line) {
            while ((getline line < contentfile) > 0) print line
            close(contentfile)
            print ""
            inserted=1
        }
        !inserted && $0 == "## Global Flags" { print_section() }
        !inserted && /^### / {
            name = substr($0, 5)
            if (index(cmdset, " " name " ") > 0 && name > newcmd) print_section()
        }
        { print }
    ' "$src"
}

# --- Pass 1: add sections for any missing commands (alphabetical order) -------
cp "$README" "$TMPFILE"
while IFS= read -r cmd; do
    if ! grep -qF "<code>sd $cmd --help</code>" "$TMPFILE"; then
        echo "Adding missing section for '$cmd'."
        insert_section "$cmd" "$TMPFILE" > "$INSERTFILE"
        mv "$INSERTFILE" "$TMPFILE"
    fi
done < <(get_command_names)

# --- Pass 2: refresh help blocks and the Global Flags section -----------------
#
# States:
#   normal            - default, copying lines through
#   wait_for_open     - saw a <summary>sd CMD --help</summary>, waiting for opening ```
#   skip_cmd_block    - inside old help code block, skipping until closing ```
#   skip_global_block - inside old Global Flags code block, skipping until closing ```
state="normal"
current_cmd=""
saw_global_heading=false

while IFS= read -r line || [[ -n "$line" ]]; do
    case "$state" in
        normal)
            printf '%s\n' "$line"

            # Detect command help summary tag
            if [[ "$line" =~ \<summary\>\<code\>sd\ (.+)\ --help\</code\>\</summary\> ]]; then
                current_cmd="${BASH_REMATCH[1]}"
                state="wait_for_open"

            # Detect the Global Flags heading
            elif [[ "$line" == "## Global Flags" ]]; then
                saw_global_heading=true

            # Detect the code block that follows the Global Flags heading
            elif $saw_global_heading && [[ "$line" == '```' ]]; then
                printf '%s\n' "$GLOBAL_FLAGS_CONTENT"
                saw_global_heading=false
                state="skip_global_block"

            # Reset the heading flag if we hit another heading first
            elif $saw_global_heading && [[ "$line" =~ ^## ]]; then
                saw_global_heading=false
            fi
            ;;

        wait_for_open)
            if [[ "$line" == '```' ]]; then
                printf '%s\n' "$line"
                get_summarized_help "$current_cmd"
                state="skip_cmd_block"
            else
                printf '%s\n' "$line"
            fi
            ;;

        skip_cmd_block)
            if [[ "$line" == '```' ]]; then
                printf '%s\n' "$line"
                state="normal"
                current_cmd=""
            fi
            # else: skip old content
            ;;

        skip_global_block)
            if [[ "$line" == '```' ]]; then
                printf '%s\n' "$line"
                state="normal"
            fi
            # else: skip old content
            ;;
    esac
done < "$TMPFILE" > "$INSERTFILE"
mv "$INSERTFILE" "$TMPFILE"

# --- Pass 3: regenerate the "# Command Line Interface" summary table ----------
build_table > "$CONTENTFILE"
awk -v contentfile="$CONTENTFILE" '
    /^\| Command \| Description \|$/ {
        while ((getline line < contentfile) > 0) print line
        close(contentfile)
        in_table=1; next
    }
    in_table && /^\|/ { next }
    in_table          { in_table=0 }
    { print }
' "$TMPFILE" > "$INSERTFILE"
mv "$INSERTFILE" "$TMPFILE"

mv "$TMPFILE" "$README"

echo "README.md help sections updated."
