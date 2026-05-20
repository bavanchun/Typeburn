package cli

import "fmt"

const (
	ExitOK       = 0
	ExitUsage    = 1
	ExitIO       = 2
	ExitAbort    = 3
	ExitInternal = 4
)

// ExitError carries the process exit code for command failures.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

func usageError(format string, args ...any) error {
	return &ExitError{Code: ExitUsage, Err: fmt.Errorf(format, args...)}
}

func ioError(format string, args ...any) error {
	return &ExitError{Code: ExitIO, Err: fmt.Errorf(format, args...)}
}

func abortError(format string, args ...any) error {
	return &ExitError{Code: ExitAbort, Err: fmt.Errorf(format, args...)}
}

// ExitCode maps an error returned by Execute into a process exit code.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	if e, ok := err.(*ExitError); ok {
		return e.Code
	}
	return ExitUsage
}
