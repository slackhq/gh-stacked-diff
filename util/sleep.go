package util

import (
	"time"
)

var defaultSleep = time.Sleep

// Sleep function that can be overridden for testing.
// Default is [time.Sleep]
func Sleep(d time.Duration) {
	defaultSleep(d)
}

// Override the default sleep. For use by tests.
func SetDefaultSleep(newSleep func(d time.Duration)) {
	defaultSleep = newSleep
}
