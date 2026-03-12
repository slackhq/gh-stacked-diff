package util

import (
	"fmt"
	"io"
)

// println or dies.
func Fprintln(w io.Writer, a ...any) {
	if _, err := fmt.Fprintln(w, a...); err != nil {
		panic(err)
	}
}

// print or dies.
func Fprint(w io.Writer, a ...any) {
	if _, err := fmt.Fprint(w, a...); err != nil {
		panic(err)
	}
}
