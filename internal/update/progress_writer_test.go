package update

import (
	"bytes"
	"io"
	"testing"
	"time"
)

// fakeClock advances only when step is called, so throttle behaviour is tested
// deterministically instead of by sleeping.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time       { return c.t }
func (c *fakeClock) step(d time.Duration) { c.t = c.t.Add(d) }

func TestProgressWriter_CountsEveryByte(t *testing.T) {
	var sink bytes.Buffer
	pw := newProgressWriter(&sink, 12, nil)

	if _, err := io.Copy(pw, bytes.NewReader(bytes.Repeat([]byte("x"), 12))); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if pw.done != 12 {
		t.Errorf("done = %d, want 12", pw.done)
	}
	if sink.Len() != 12 {
		t.Errorf("underlying writer got %d bytes, want 12", sink.Len())
	}
}

// The size-cap and empty-download guards in downloadTo read the byte count that
// io.Copy returns, so the decorator must never short-write or under-report.
func TestProgressWriter_ReportsFullWriteLength(t *testing.T) {
	pw := newProgressWriter(io.Discard, 0, nil)
	n, err := pw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if n != 5 {
		t.Errorf("n = %d, want 5", n)
	}
}

func TestProgressWriter_ThrottlesCallbacks(t *testing.T) {
	clk := &fakeClock{t: time.Unix(0, 0)}
	var calls int
	pw := newProgressWriter(io.Discard, 100, func(_, _ int64) { calls++ })
	pw.now = clk.now

	// 100 writes inside a single interval must collapse to one callback: the
	// first write is due (last is the zero time), the rest are throttled.
	for range 100 {
		_, _ = pw.Write([]byte("x"))
	}
	if calls != 1 {
		t.Fatalf("calls within one interval = %d, want 1", calls)
	}

	clk.step(progressInterval)
	_, _ = pw.Write([]byte("x"))
	if calls != 2 {
		t.Errorf("calls after interval elapsed = %d, want 2", calls)
	}
}

// A throttled final write would otherwise leave a progress bar frozen short of
// its true end value.
func TestProgressWriter_FlushAlwaysEmits(t *testing.T) {
	clk := &fakeClock{t: time.Unix(0, 0)}
	var last struct{ done, total int64 }
	var calls int
	pw := newProgressWriter(io.Discard, 40, func(d, tot int64) {
		calls++
		last.done, last.total = d, tot
	})
	pw.now = clk.now

	_, _ = pw.Write(bytes.Repeat([]byte("x"), 40))
	before := calls
	pw.flush()

	if calls != before+1 {
		t.Errorf("flush did not emit: calls %d → %d", before, calls)
	}
	if last.done != 40 || last.total != 40 {
		t.Errorf("final report = (%d,%d), want (40,40)", last.done, last.total)
	}
}

func TestProgressWriter_NilCallbackIsSilent(t *testing.T) {
	pw := newProgressWriter(io.Discard, 10, nil)
	if _, err := pw.Write([]byte("data")); err != nil {
		t.Fatalf("write: %v", err)
	}
	pw.flush() // must not panic
}

// A response without Content-Length reports -1; callers need a single sentinel
// meaning "unknown", so it is normalised to 0.
func TestProgressWriter_UnknownTotalNormalisedToZero(t *testing.T) {
	var got int64 = -1
	pw := newProgressWriter(io.Discard, -1, func(_, total int64) { got = total })
	pw.flush()
	if got != 0 {
		t.Errorf("total = %d, want 0", got)
	}
}
