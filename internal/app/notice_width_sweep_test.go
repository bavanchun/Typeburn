package app

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestView_NoticeNeverWidensTheFrame sweeps every column from the supported
// minimum to 90 with a notice present.
//
// The notice is not drawn beside the frame, it is written into it, and the
// frame is then padded to its own widest line. So a notice one cell too wide
// does not clip — it pushes every row of the screen out with it. Sampling round
// widths missed this: the full notice is 78 cells and only 60, 61 and 72 could
// not hold it.
func TestView_NoticeNeverWidensTheFrame(t *testing.T) {
	const reason = "could not write history.json: permission denied"

	for w := 60; w <= 90; w++ {
		m := baseModel(w, 24)
		m.persistErr = reason

		for i, ln := range strings.Split(stripANSI(m.View().Content), "\n") {
			if got := lipgloss.Width(ln); got > w {
				t.Errorf("width %d: line %d is %d cells: %q", w, i, got, ln)
			}
		}
	}
}
