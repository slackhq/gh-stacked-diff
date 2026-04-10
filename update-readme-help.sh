#!/bin/bash
# update-readme-help.sh
#
# Updates the --help sections in README.md to match the current command output.
# The Global Flags in each command section are replaced with a short summary
# that references the full Global Flags section at the bottom of the README.
#
# Usage: ./update-readme-help.sh

set -euo pipefail

cd "$(dirname "$0")"

BINARY="./bin/gh-stacked-diff"
README="README.md"

make build

TMPFILE=$(mktemp)
trap 'rm -f "$TMPFILE"' EXIT

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

# Process README.md with a state machine.
#
# States:
#   normal           - default, copying lines through
#   wait_for_open    - saw a <summary>sd CMD --help</summary>, waiting for opening ```
#   skip_cmd_block   - inside old help code block, skipping until closing ```
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
done < "$README" > "$TMPFILE"

mv "$TMPFILE" "$README"

echo "README.md help sections updated."
