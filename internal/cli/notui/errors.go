package notui

import "errors"

var (
	ErrAbort       = errors.New("typing test aborted")
	ErrNotTerminal = errors.New("stdin is not a terminal")
)
