package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequireGitRef_ValidRefs(t *testing.T) {
	// Should not panic
	RequireGitRef("main")
	RequireGitRef("feature/my-branch")
	RequireGitRef("abc123")
	RequireGitRef("v1.0.0")
}

func TestRequireGitRef_EmptyString(t *testing.T) {
	assert.PanicsWithValue(t, "expected git ref, got empty string", func() {
		RequireGitRef("")
	})
}

func TestRequireGitRef_StartsWithHyphen(t *testing.T) {
	assert.PanicsWithValue(t, "git ref must not start with '-': --upload-pack=evil", func() {
		RequireGitRef("--upload-pack=evil")
	})
}

func TestRequireGitRef_SingleHyphen(t *testing.T) {
	assert.PanicsWithValue(t, "git ref must not start with '-': -branch", func() {
		RequireGitRef("-branch")
	})
}
