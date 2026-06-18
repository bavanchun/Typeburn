package anim

import "testing"

func TestClockActiveTransitions(t *testing.T) {
	var c Clock
	c.Add(Tween{StartMs: 0, DurMs: 100})
	c.Add(Tween{StartMs: 50, DurMs: 100}) // ends at 150

	if !c.Active(0) {
		t.Errorf("Active(0) false, want true")
	}
	if !c.Active(120) { // first done, second still live
		t.Errorf("Active(120) false, want true")
	}
	if c.Active(150) { // both done
		t.Errorf("Active(150) true, want false")
	}
}

func TestClockEmptyInactive(t *testing.T) {
	var c Clock
	if c.Active(0) {
		t.Errorf("empty clock should be inactive")
	}
}

func TestClockPrune(t *testing.T) {
	var c Clock
	c.Add(Tween{StartMs: 0, DurMs: 100}) // done at 100
	c.Add(Tween{StartMs: 0, DurMs: 500}) // done at 500
	c.Prune(200)
	if c.Len() != 1 {
		t.Errorf("after prune Len=%d want 1", c.Len())
	}
	if !c.Active(200) {
		t.Errorf("surviving tween should keep clock active")
	}
	c.Prune(600)
	if c.Len() != 0 {
		t.Errorf("after full prune Len=%d want 0", c.Len())
	}
}
