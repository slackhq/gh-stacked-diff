package util

import (
	"regexp"
	"strconv"
	"time"
)

// Number of times to retry a "gh" command that fails due to network errors.
const GhRetries = 2
const RetryDelay = 1 * time.Second

// Number of times to retry a "git" command that fails due to ".git/index.lock" contention.
const IndexLockRetries = 5

// Matches git's transient "index.lock" contention error, which is safe to retry.
var indexLockRegexp = regexp.MustCompile(`Unable to create '.*index\.lock': File exists\.`)

// Matches gh's HTTP error formats: "gh: Not Found (HTTP 404)" or "gh: HTTP 304".
var ghHTTPErrorRegexp = regexp.MustCompile(`HTTP (\d+)`)

// RetryOnIndexLock retries when a git command fails due to transient
// ".git/index.lock" contention, up to [IndexLockRetries] times.
func RetryOnIndexLock(executionCount int, stderr string) bool {
	retry := executionCount <= IndexLockRetries && indexLockRegexp.MatchString(stderr)
	if retry {
		Sleep(RetryDelay)
	}
	return retry
}

// RetryGhOnNetworkError retries gh commands on network errors and retryable HTTP
// errors (5xx, 408, etc.). Does not retry on 3xx or 429 (rate limit).
func RetryGhOnNetworkError(executionCount int, stderr string) bool {
	if executionCount > GhRetries {
		return false
	}
	if match := ghHTTPErrorRegexp.FindStringSubmatch(stderr); match != nil {
		code, _ := strconv.Atoi(match[1])
		if (code >= 300 && code < 400) || code == 429 {
			return false
		}
	}
	Sleep(time.Duration(executionCount) * RetryDelay)
	return true
}

// RetryUpTo returns a [ShouldRetryFunc] that retries on any failure, up to maxRetries times.
func RetryUpTo(maxRetries int) ShouldRetryFunc {
	return func(executionCount int, _ string) bool {
		retry := executionCount <= maxRetries
		if retry {
			Sleep(time.Duration(executionCount) * RetryDelay)
		}
		return retry
	}
}
