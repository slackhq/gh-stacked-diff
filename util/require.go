package util

import "regexp"

func RequireNotEmptyString(s string) {
	if s == "" {
		panic("Missing value")
	}
}

var hexPattern = regexp.MustCompile(`^[0-9a-fA-F]+$`)

// RequireHexString panics if s is not a non-empty hexadecimal string.
// Use this to validate git commit hashes before interpolating them into
// jq filters or other query strings.
func RequireHexString(s string) {
	if !hexPattern.MatchString(s) {
		panic("expected hex string, got: " + s)
	}
}

// RequireGitRef panics if s is empty or starts with a hyphen.
// Git interprets arguments starting with "-" as flags, so passing an
// unvalidated ref (branch name, commit hash) from an external source
// (e.g. GitHub API) could cause git to treat it as an option.
func RequireGitRef(s string) {
	if s == "" {
		panic("expected git ref, got empty string")
	}
	if s[0] == '-' {
		panic("git ref must not start with '-': " + s)
	}
}

var hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]*$`)

// RequireHostname panics if s contains characters outside the set allowed
// in hostnames (alphanumeric, dots, hyphens, underscores). Empty strings
// are allowed. Use this to validate hostnames before interpolating them
// into jq filters or other query strings.
func RequireHostname(s string) {
	if !hostnamePattern.MatchString(s) {
		panic("expected hostname, got: " + s)
	}
}
