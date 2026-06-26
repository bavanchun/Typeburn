package notui

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/runner"
)

// testClock is a deterministic monotonic clock for tests.
// Each now() call returns the current value then advances by stepMs.
// Start at a non-zero value: Engine uses 0 as the "not yet started" sentinel
// for StartMs, so a first-keystroke timestamp of 0 causes the engine to treat
// the first rune as "no key pressed", breaking timer-arm and endMs math.
type testClock struct {
	curMs  int64
	stepMs int64
}

func newTestClock(startMs, stepMs int64) *testClock {
	return &testClock{curMs: startMs, stepMs: stepMs}
}

func (c *testClock) now() time.Time {
	t := time.UnixMilli(c.curMs)
	c.curMs += c.stepMs
	return t
}

// timerFactory produces a single controllable timer channel. Calling fire()
// unblocks the timer case in runLoop synchronously — no real sleep needed.
type timerFactory struct{ ch chan time.Time }

func newTimerFactory() *timerFactory { return &timerFactory{ch: make(chan time.Time, 1)} }

func (f *timerFactory) after(_ time.Duration) <-chan time.Time { return f.ch }
func (f *timerFactory) fire()                                  { f.ch <- time.Time{} }

// makeTimeSession creates a Time-mode session with a long buffer so keystrokes
// never exhaust the target within the test window.
func makeTimeSession(lengthSec int) runner.Session {
	tgt := strings.Repeat("hello world ", 500)
	eng := runner.RebuildEngine(tgt, config.ModeTime, lengthSec, false)
	return runner.Session{Engine: eng, Target: tgt, Mode: config.ModeTime, Length: lengthSec}
}

func makeWordsSession(tgt string, wordCount int) runner.Session {
	eng := runner.RebuildEngine(tgt, config.ModeWords, wordCount, false)
	return runner.Session{Engine: eng, Target: tgt, Mode: config.ModeWords, Length: wordCount}
}

// runTimerTest drives the select loop synchronously: it feeds each rune through
// an unbuffered events channel (so every send blocks until drive has consumed
// it), then fires the timer. Because the last send returns only after drive has
// received and applied the final keystroke, the subsequent timer fire is the
// only ready select case — fully deterministic, no pipe, no reader goroutine,
// no real sleeps.
func runTimerTest(t *testing.T, sess runner.Session, input string, clk *testClock, tf *timerFactory) metrics.Result {
	t.Helper()

	rCh := make(chan metrics.Result, 1)
	captureWrite := ResultWriter(func(_ io.Writer, r metrics.Result) error {
		rCh <- r
		return nil
	})

	events := make(chan readMsg) // unbuffered: each send blocks until consumed
	go func() {
		_ = drive(io.Discard, sess, captureWrite, clk.now, tf.after, events)
	}()

	for _, r := range input {
		events <- readMsg{ev: Event{Kind: EventRune, Rune: r}}
	}
	tf.fire()

	select {
	case r := <-rCh:
		return r
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for result")
		return metrics.Result{}
	}
}

// TestRunLoop_TimeMode_AutoEnd_NoAFK: Time/30s, last keystroke within 7s of
// the clock limit — AFKTrim does NOT fire. Timer fires → DurationMs == Length*1000.
//
// Clock: startMs=1000, stepMs=6000. 5 runes land at 1000..25000.
// endMs = 1000+30000 = 31000. lastKeyMs=25000. gap=6000 ≤ 7000 → no trim.
// DurationMs = 31000-1000 = 30000.
func TestRunLoop_TimeMode_AutoEnd_NoAFK(t *testing.T) {
	const lengthSec = 30
	clk := newTestClock(1000, 6000)
	result := runTimerTest(t, makeTimeSession(lengthSec), "hello", clk, newTimerFactory())
	if result.DurationMs != int64(lengthSec*1000) {
		t.Errorf("DurationMs: want %d, got %d", lengthSec*1000, result.DurationMs)
	}
	if result.CorrectChars+result.IncorrectChars == 0 {
		t.Error("expected non-zero char counts")
	}
}

// TestRunLoop_TimeMode_AutoEnd_WithAFK: Time/30s, runes end well before the
// window (>7s trailing gap). Timer fires → DurationMs == lastKeyMs-startMs
// (AFK-trimmed), NOT Length*1000. This is the core TUI parity assertion.
//
// Clock: startMs=1000, stepMs=100. 3 runes at 1000,1100,1200.
// endMs=31000. gap=29800 > 7000 → trim fires → effective endMs=1200.
// DurationMs = 1200-1000 = 200.
func TestRunLoop_TimeMode_AutoEnd_WithAFK(t *testing.T) {
	const lengthSec = 30
	clk := newTestClock(1000, 100)
	result := runTimerTest(t, makeTimeSession(lengthSec), "hel", clk, newTimerFactory())
	const wantMs = int64(200)
	if result.DurationMs != wantMs {
		t.Errorf("DurationMs: want %d (AFK-trimmed), got %d", wantMs, result.DurationMs)
	}
	if result.DurationMs == int64(lengthSec*1000) {
		t.Error("DurationMs equals full window — AFK trim should have fired")
	}
}

// TestRunLoop_WordsMode_EventDriven: Words mode completes via engine event;
// no timer required. Regression guard for non-Time paths.
func TestRunLoop_WordsMode_EventDriven(t *testing.T) {
	tgt := "hello world"
	sess := makeWordsSession(tgt, 2)
	clk := newTestClock(1000, 50)

	pr, pw := io.Pipe()
	go func() { _, _ = pw.Write([]byte(tgt)); pw.Close() }()

	var result metrics.Result
	err := runLoop(pr, io.Discard, sess,
		func(_ io.Writer, r metrics.Result) error { result = r; return nil },
		clk.now, newTimerFactory().after,
	)
	if err != nil {
		t.Fatalf("runLoop error: %v", err)
	}
	if result.CorrectChars == 0 {
		t.Error("expected correct chars")
	}
	if result.DurationMs <= 0 {
		t.Error("expected positive DurationMs")
	}
}

// TestRunLoop_BackspaceOnly_NoChars: backspace-only input on empty buffer (no-op
// in the engine). EOF terminates the loop; no panic; empty log gives Accuracy=100.
func TestRunLoop_BackspaceOnly_NoChars(t *testing.T) {
	clk := newTestClock(1000, 100)
	sess := makeTimeSession(30)

	pr, pw := io.Pipe()
	go func() { _, _ = pw.Write([]byte{0x7f, 0x7f, 0x7f}); pw.Close() }()

	err := runLoop(pr, io.Discard, sess, nil, clk.now, newTimerFactory().after)
	if err != nil && err != io.EOF && err != io.ErrClosedPipe {
		t.Fatalf("unexpected error: %v", err)
	}
	z := metrics.Compute(sess.Engine.Log(), sess.Mode, 0)
	if z.Accuracy != 100 {
		t.Errorf("empty log accuracy: want 100, got %v", z.Accuracy)
	}
}
