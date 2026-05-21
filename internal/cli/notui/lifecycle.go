package notui

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

type rawOps struct {
	isTerminal  func(int) bool
	makeRaw     func(int) (*term.State, error)
	restore     func(int, *term.State) error
	notify      func(chan<- os.Signal, ...os.Signal)
	stop        func(chan<- os.Signal)
	afterNotify func(chan os.Signal)
}

func defaultRawOps() rawOps {
	return rawOps{
		isTerminal: term.IsTerminal,
		makeRaw:    term.MakeRaw,
		restore:    term.Restore,
		notify:     signal.Notify,
		stop:       signal.Stop,
	}
}

func runRaw(ctx context.Context, in *os.File, out io.Writer, fn func(io.Reader, io.Writer) error) error {
	fd := int(in.Fd())
	if !term.IsTerminal(fd) {
		return ErrNotTerminal
	}
	return runRawWith(ctx, fd, in, out, defaultRawOps(), fn)
}

func runRawWith(
	ctx context.Context,
	fd int,
	in io.Reader,
	out io.Writer,
	ops rawOps,
	fn func(io.Reader, io.Writer) error,
) (err error) {
	sigCh := make(chan os.Signal, 1)
	ops.notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer ops.stop(sigCh)
	if ops.afterNotify != nil {
		ops.afterNotify(sigCh)
	}

	old, err := ops.makeRaw(fd)
	if err != nil {
		return err
	}
	restored := false
	restore := func() {
		if !restored {
			_ = ops.restore(fd, old)
			restored = true
		}
	}
	defer restore()

	done := make(chan rawResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- rawResult{panicValue: r}
			}
		}()
		done <- rawResult{err: fn(in, out)}
	}()

	select {
	case <-ctx.Done():
		restore()
		return ctx.Err()
	case res := <-done:
		restore()
		if res.panicValue != nil {
			panic(res.panicValue)
		}
		return res.err
	case sig := <-sigCh:
		restore()
		if sig == syscall.SIGINT {
			return ErrAbort
		}
		return sigError(sig)
	}
}

type rawResult struct {
	err        error
	panicValue any
}

func sigError(sig os.Signal) error {
	return &signalError{sig: sig}
}

type signalError struct {
	sig os.Signal
}

func (e *signalError) Error() string {
	return "received signal: " + e.sig.String()
}
