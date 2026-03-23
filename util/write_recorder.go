package util

import (
	"bytes"
	"io"
	"sync"
)

// An [io.Writer] that outputs to a string and also to Stdout.
// Useful for testing so that log output can still be seen in the output of
// the test if the test failed, and the output of the program can be asserted
// against. Safe for concurrent use.
type WriteRecorder struct {
	// All output is written here.
	out io.Writer
	// All output is saved here.
	buffer *bytes.Buffer
	mu     sync.Mutex
}

var _ io.Writer = new(WriteRecorder)

// Creates a new [WriteRecorder] that writes to Stdout.
func NewWriteRecorder(stdout io.Writer) *WriteRecorder {
	recorder := new(WriteRecorder)
	recorder.out = stdout
	recorder.buffer = new(bytes.Buffer)
	return recorder
}

func (r *WriteRecorder) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buffer.Write(p)
	return r.out.Write(p)
}

func (r *WriteRecorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buffer.String()
}
