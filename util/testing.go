package util

import "testing"

var testInitHooks []func(t *testing.T)

// RegisterTestInitHook registers a function to be called during test initialization.
// This allows packages like interactive and gitutil to register hooks without creating import cycles.
// Hooks must be idempotent as they may be called multiple times per test.
func RegisterTestInitHook(hook func(t *testing.T)) {
	testInitHooks = append(testInitHooks, hook)
}

// RunTestInitHooks calls all registered test initialization hooks.
func RunTestInitHooks(t *testing.T) {
	for _, hook := range testInitHooks {
		hook(t)
	}
}
