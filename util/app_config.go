package util

import (
	"io"
)

// PromptForReviewType controls whether and how the user is prompted to mark a PR as ready for review.
type PromptForReviewType string

const (
	PromptForReviewNever   PromptForReviewType = "never"
	PromptForReviewPromptY PromptForReviewType = "promptY"
	PromptForReviewPromptN PromptForReviewType = "promptN"
)

func (t PromptForReviewType) IsValid() bool {
	switch t {
	case PromptForReviewNever, PromptForReviewPromptY, PromptForReviewPromptN:
		return true
	}
	return false
}

// Allows unit testing the use of standard i/o.
type StdIo struct {
	Out io.Writer
	Err io.Writer
	In  io.Reader
}

// Config to help with unit testing the app.
// For example, it allows testing code paths that would otherwise call os.Exit().
type AppConfig struct {
	Io            StdIo
	AppExecutable string         // Path of this executable.
	Exit          func(code int) // Call os.Exit with the given code, or panic during unit tests.
	UserCacheDir  string         // os.UserCacheDir or a dir specific for each test in unit tests.
	DemoMode      bool
}
