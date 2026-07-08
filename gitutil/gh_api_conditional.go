package gitutil

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

// fetchPRWithETag queries a PR using a conditional HTTP request. Pass an empty
// etag to seed the cache. Returns a new ETag if the PR changed, or the same
// etag back if unchanged (304) or on error.
func fetchPRWithETag(prNumber int, etag string) string {
	nameWithOwner := GetRepoNameWithOwner()
	endpoint := fmt.Sprintf("/repos/%s/pulls/%d", nameWithOwner, prNumber)

	args := []any{"api", "--method", "HEAD", "--include", "--hostname", GetRepoHostname()}
	if etag != "" {
		args = append(args, "-H", "If-None-Match: "+etag)
	}
	args = append(args, endpoint)

	out, execErr := util.Execute(util.ExecuteOptions{}, "gh", args...)
	if out == "" && execErr != nil {
		slog.Debug(fmt.Sprint("ETag conditional request failed: ", execErr))
		return etag
	}
	statusCode, newETag, parseErr := parseGhApiIncludeResponse(out)
	if parseErr != nil {
		slog.Debug(fmt.Sprint("failed to parse gh api --include response: ", parseErr))
		return etag
	}
	if statusCode == 304 || newETag == "" {
		return etag
	}
	return newETag
}

func parseGhApiIncludeResponse(raw string) (statusCode int, etag string, err error) {
	headerEnd := strings.Index(raw, "\r\n\r\n")
	if headerEnd == -1 {
		headerEnd = strings.Index(raw, "\n\n")
	}
	if headerEnd == -1 {
		return 0, "", fmt.Errorf("could not find header/body separator in gh api --include output")
	}

	headerSection := raw[:headerEnd]
	lines := strings.Split(headerSection, "\n")
	if len(lines) == 0 {
		return 0, "", fmt.Errorf("empty header section")
	}

	fields := strings.Fields(lines[0])
	if len(fields) < 2 {
		return 0, "", fmt.Errorf("malformed status line: %s", lines[0])
	}
	statusCode, err = strconv.Atoi(fields[1])
	if err != nil {
		return 0, "", fmt.Errorf("could not parse status code from: %s", lines[0])
	}

	for _, line := range lines[1:] {
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(key), "etag") {
			etag = strings.TrimSpace(value)
			break
		}
	}

	return statusCode, etag, nil
}
