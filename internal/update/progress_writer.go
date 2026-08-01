package update

import (
	"io"
	"time"
)

// progressInterval is the minimum gap between byte-progress callbacks. A ~4 MB
// archive arrives in thousands of small writes; without throttling every one of
// them would cross into the front-end's render loop. 50ms is well below the
// perceptual threshold for a progress bar while bounding a 4 MB download to
// roughly a hundred callbacks.
const progressInterval = 50 * time.Millisecond

// progressWriter decorates a writer, counting bytes and forwarding throttled
// updates to onBytes. It is a plain io.Writer with no goroutines: the caller's
// io.Copy drives it, so it inherits that call's synchronisation.
//
// Write is a faithful pass-through: it returns the underlying writer's own n
// and error untouched, adding only the byte counting. That is what keeps the
// count io.Copy returns exactly as meaningful as before — the empty-download
// and size-cap guards in downloadTo are computed from it.
type progressWriter struct {
	w       io.Writer
	onBytes func(done, total int64)
	total   int64
	done    int64
	last    time.Time
	now     func() time.Time // swappable for deterministic tests
}

// newProgressWriter wraps w. A nil onBytes yields a writer that only counts,
// so callers never need a guard of their own.
func newProgressWriter(w io.Writer, total int64, onBytes func(done, total int64)) *progressWriter {
	// A negative Content-Length means "unknown"; normalise it to 0 so callers
	// have a single sentinel to test for.
	if total < 0 {
		total = 0
	}
	return &progressWriter{w: w, onBytes: onBytes, total: total, now: time.Now}
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	p.done += int64(n)

	if p.onBytes != nil {
		if t := p.now(); t.Sub(p.last) >= progressInterval {
			p.last = t
			p.onBytes(p.done, p.total)
		}
	}
	return n, err
}

// flush emits a final update regardless of the throttle, so a bar always lands
// on its true end value instead of freezing wherever the last tick left it.
// Safe to call on a writer that never saw a byte.
func (p *progressWriter) flush() {
	if p.onBytes != nil {
		p.onBytes(p.done, p.total)
	}
}
