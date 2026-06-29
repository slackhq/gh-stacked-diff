package util

import (
	"regexp"
	"time"
)

// Number of times to retry a "gh" command that fails. GitHub commands can fail
// transiently, so they are retried on any error.
const GhRetries = 2
const RetryDelay = 1 * time.Second

// Number of times to retry a "git" command that fails due to ".git/index.lock" contention.
const IndexLockRetries = 5

// Matches git's transient "index.lock" contention error, which is safe to retry.
var indexLockRegexp = regexp.MustCompile(`Unable to create '.*index\.lock': File exists\.`)

// RetryOnIndexLock retries when a git command fails due to transient
// ".git/index.lock" contention, up to [IndexLockRetries] times.
func RetryOnIndexLock(executionCount int, stderr string) bool {
	retry := executionCount <= IndexLockRetries && indexLockRegexp.MatchString(stderr)
	if retry {
		Sleep(RetryDelay)
	}
	return retry
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
