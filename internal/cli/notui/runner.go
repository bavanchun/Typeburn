package notui

import (
	"bufio"
	"context"
	"io"
	"os"
	"time"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/runner"
)

type ResultWriter func(io.Writer, metrics.Result) error

func Run(ctx context.Context, in *os.File, out io.Writer, session runner.Session, write ResultWriter) error {
	return runRaw(ctx, in, out, func(r io.Reader, w io.Writer) error {
		return runLoop(r, w, session, write, time.Now, time.After)
	})
}

// readMsg carries one ReadEvent result from the reader goroutine to runLoop.
// Errors (including EOF and ErrAbort-carrying EventAbort) terminate the loop.
type readMsg struct {
	ev  Event
	err error
}

// runLoop drives the typing session to completion.
//
// Concurrency invariants:
//   - session.Engine is mutated and read ONLY in this goroutine.
//   - The reader goroutine touches only the bufio.Reader and the send channel.
//   - All output (RenderPrompt/RenderStatus/RenderSummary/write) happens here.
//   - The reader goroutine MUST NOT close the channel; runLoop only receives.
//     A single blocked reader may remain after return — intentionally abandoned
//     at process teardown, identical to the runRaw done-goroutine in lifecycle.go.
//
// Timer seam: `after` defaults to time.After in Run; tests inject a controlled
// channel so the timer fires synchronously without real sleeps.
func runLoop(
	in io.Reader,
	out io.Writer,
	session runner.Session,
	write ResultWriter,
	now func() time.Time,
	after func(time.Duration) <-chan time.Time,
) error {
	RenderPrompt(out, session.Target)
	rd := bufio.NewReader(in)

	// Cap-1 buffer: lets the reader goroutine deposit one message after drive
	// returns without blocking, preventing a goroutine leak at process teardown.
	events := make(chan readMsg, 1)
	go func() {
		for {
			ev, err := ReadEvent(rd)
			events <- readMsg{ev: ev, err: err}
			if err != nil {
				return // reader goroutine exits; never closes the channel
			}
		}
	}()

	return drive(out, session, write, now, after, events)
}

// drive runs the event/timer select loop, consuming readMsgs from events.
// Splitting it out of runLoop lets tests feed events synchronously through an
// unbuffered channel (no pipe, no reader goroutine, no real sleeps) while the
// production path supplies the reader-fed cap-1 channel built in runLoop.
func drive(
	out io.Writer,
	session runner.Session,
	write ResultWriter,
	now func() time.Time,
	after func(time.Duration) <-chan time.Time,
	events <-chan readMsg,
) error {
	// timer is nil until the first keystroke in Time mode arms it. Lazily arming
	// on the first-keystroke transition (StartMs 0 → non-zero) makes the timer
	// fire instant structurally equal to StartMs + Length*1000 — no desync.
	// Using a nil channel in select is safe: a nil case is never selected.
	var timer <-chan time.Time
	timerArmed := false

	// Zero-input path: if no keystroke ever arrives the timer is never armed
	// and runLoop blocks on the events channel until EOF or SIGINT (ErrAbort).
	// This is the accepted "no input, no test" behavior; no separate idle timer.

	for {
		select {
		case msg := <-events:
			if msg.err != nil {
				return msg.err
			}
			nowMs := now().UnixMilli()
			switch msg.ev.Kind {
			case EventNone:
				continue // no-op key (e.g. arrow/escape seq); skip repaint
			case EventAbort:
				return ErrAbort
			case EventBackspace:
				session.Engine.Backspace(nowMs)
			case EventRune:
				session.Engine.Apply(msg.ev.Rune, nowMs)
			}

			// Arm the timer on the keystroke that starts the clock (StartMs 0 → non-zero).
			// Only relevant for Time mode; for other modes timer stays nil forever.
			if !timerArmed && session.Mode == config.ModeTime && session.Engine.StartMs() != 0 {
				timer = after(time.Duration(session.Length) * time.Second)
				timerArmed = true
			}

			done, total := session.Engine.Progress()
			elapsed := nowMs - session.Engine.StartMs()
			live := metrics.LiveWPM(session.Engine.Log(), elapsed)
			RenderStatus(out, done, total, live)

			// Event-driven completion: Words/Quote/Code use engine completion;
			// Time mode can also complete here (backward compat / pre-timer edge).
			if completed(session, nowMs) {
				result := metrics.Compute(session.Engine.Log(), session.Mode, endMs(session, nowMs))
				if write != nil {
					return write(out, result)
				}
				RenderSummary(out, result)
				return nil
			}

		case <-timer:
			// Time mode: clock expired. No trailing keystrokes can enter the log
			// after this point because we return immediately. Compute uses
			// StartMs + Length*1000 as endMs; TrimAFK may reduce it if trailing
			// idle exceeds 7s — that is intended parity with the TUI path.
			// RenderStatus is intentionally skipped here to avoid a stale frame.
			// endMs ignores its nowMs arg in Time mode (returns StartMs + Length*1000),
			// which is the only mode that arms the timer, so pass 0.
			result := metrics.Compute(session.Engine.Log(), session.Mode, endMs(session, 0))
			if write != nil {
				return write(out, result)
			}
			RenderSummary(out, result)
			return nil
		}
	}
}

func completed(session runner.Session, nowMs int64) bool {
	if session.Mode == config.ModeTime {
		start := session.Engine.StartMs()
		return start > 0 && nowMs-start >= int64(session.Length*1000)
	}
	return session.Engine.Complete(nowMs)
}

func endMs(session runner.Session, nowMs int64) int64 {
	if session.Mode == config.ModeTime {
		return session.Engine.StartMs() + int64(session.Length*1000)
	}
	return nowMs
}
