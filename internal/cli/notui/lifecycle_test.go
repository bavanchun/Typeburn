package notui

import (
	"context"
	"io"
	"os"
	"syscall"
	"testing"

	"golang.org/x/term"
)

func TestRunRawWithRestoresOnReturn(t *testing.T) {
	restores := 0
	err := runRawWith(context.Background(), 1, nil, io.Discard, testOps(&restores, nil), func(io.Reader, io.Writer) error {
		return nil
	})
	if err != nil {
		t.Fatalf("runRawWith: %v", err)
	}
	if restores != 1 {
		t.Fatalf("restore count = %d", restores)
	}
}

func TestRunRawWithRestoresOnPanic(t *testing.T) {
	restores := 0
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
		if restores != 1 {
			t.Fatalf("restore count = %d", restores)
		}
	}()
	_ = runRawWith(context.Background(), 1, nil, io.Discard, testOps(&restores, nil), func(io.Reader, io.Writer) error {
		panic("boom")
	})
}

func TestRunRawWithRestoresOnSIGINT(t *testing.T) {
	restores := 0
	err := runRawWith(context.Background(), 1, nil, io.Discard, testOps(&restores, func(ch chan os.Signal) {
		ch <- syscall.SIGINT
	}), func(io.Reader, io.Writer) error {
		select {}
	})
	if err != ErrAbort {
		t.Fatalf("want abort, got %v", err)
	}
	if restores != 1 {
		t.Fatalf("restore count = %d", restores)
	}
}

func testOps(restores *int, after func(chan os.Signal)) rawOps {
	return rawOps{
		isTerminal: func(int) bool { return true },
		makeRaw:    func(int) (*term.State, error) { return &term.State{}, nil },
		restore: func(int, *term.State) error {
			*restores++
			return nil
		},
		notify:      func(chan<- os.Signal, ...os.Signal) {},
		stop:        func(chan<- os.Signal) {},
		afterNotify: after,
	}
}
