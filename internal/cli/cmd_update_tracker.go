package cli

import (
	"sync"

	"github.com/bavanchun/Typeburn/v2/internal/update"
)

// tracker is the hand-off between the blocking Apply goroutine and the render
// loop. Progress is state, not an event stream, so the renderer samples the
// latest snapshot on its own cadence instead of draining a channel: no
// backpressure on the download, and no stage transition can be dropped.
type tracker struct {
	mu  sync.Mutex
	cur update.Progress
}

func (t *tracker) set(p update.Progress) {
	t.mu.Lock()
	t.cur = p
	t.mu.Unlock()
}

func (t *tracker) get() update.Progress {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cur
}
