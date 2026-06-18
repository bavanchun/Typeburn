package anim

// Clock aggregates a set of tweens and reports whether any are still running.
// The frame driver polls Active(nowMs) to decide whether to re-arm its tick, so
// the animation loop self-stops once every tween has completed.
type Clock struct {
	tweens []Tween
}

// Add registers a tween with the clock.
func (c *Clock) Add(tw Tween) { c.tweens = append(c.tweens, tw) }

// Active reports whether at least one tween is not yet done at nowMs.
func (c *Clock) Active(nowMs int64) bool {
	for _, tw := range c.tweens {
		if !tw.Done(nowMs) {
			return true
		}
	}
	return false
}

// Prune drops every tween that has completed at nowMs, keeping the slice small
// for long-lived clocks. Order of remaining tweens is preserved.
func (c *Clock) Prune(nowMs int64) {
	kept := c.tweens[:0]
	for _, tw := range c.tweens {
		if !tw.Done(nowMs) {
			kept = append(kept, tw)
		}
	}
	c.tweens = kept
}

// Len returns the number of tweens currently tracked (after any pruning).
func (c *Clock) Len() int { return len(c.tweens) }
