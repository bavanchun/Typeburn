package app

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// noticeRow returns the index of the row carrying want, or -1.
func noticeRow(frame, want string) int {
	for i, ln := range strings.Split(stripANSI(frame), "\n") {
		if strings.Contains(ln, want) {
			return i
		}
	}
	return -1
}

// TestOverlayNotice_LandsOnARowTheTerminalShows is the whole point of the
// placement: a notice written to the frame's last line disappears the moment
// the frame is taller than the terminal, which is precisely when there is
// something worth saying.
func TestOverlayNotice_LandsOnARowTheTerminalShows(t *testing.T) {
	const notice = "something to say"

	for _, tc := range []struct {
		name  string
		frame string
		w, h  int
	}{
		{"frame taller than the terminal", strings.Repeat("content\n", 28) + "content", 80, 24},
		{"frame exactly the terminal height", strings.Repeat("content\n", 23) + "content", 80, 24},
		{"frame shorter than the terminal", strings.Repeat("content\n", 9) + "content", 80, 24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := overlayNotice(tc.frame, notice, tc.w, tc.h)
			row := noticeRow(out, notice)
			if row < 0 {
				t.Fatalf("the notice is not in the frame at all:\n%s", out)
			}
			if row > tc.h-1 {
				t.Errorf("notice on row %d of a %d-row terminal — clipped away", row, tc.h)
			}
		})
	}
}

// TestOverlayNotice_DoesNotOverwriteContent: when the chosen row carries
// something, the notice is inserted rather than written over it. The footer
// keybindings are not an acceptable price for a toast.
func TestOverlayNotice_DoesNotOverwriteContent(t *testing.T) {
	frame := strings.Join([]string{"one", "two", "footer"}, "\n")

	out := overlayNotice(frame, "notice", 20, 10)

	for _, want := range []string{"one", "two", "footer", "notice"} {
		if !strings.Contains(stripANSI(out), want) {
			t.Errorf("%q was lost:\n%s", want, stripANSI(out))
		}
	}
}

// TestOverlayNotice_TakesABlankRowWithoutGrowingTheFrame: the common case is a
// frame padded to the terminal height, where the notice costs nothing.
func TestOverlayNotice_TakesABlankRowWithoutGrowingTheFrame(t *testing.T) {
	frame := strings.Join([]string{"one", "two", "footer", "    "}, "\n")

	out := overlayNotice(frame, "notice", 20, 4)

	if got := len(strings.Split(out, "\n")); got != 4 {
		t.Errorf("frame grew to %d rows; the blank padding row was free", got)
	}
	if row := noticeRow(out, "notice"); row != 3 {
		t.Errorf("notice on row %d, want the blank row 3", row)
	}
}

// TestOverlayNotice_DoesNotWidenAFrameThatFits guards the horizontal axis: a
// notice narrower than the terminal must not push the frame wider than the
// content it joins.
func TestOverlayNotice_DoesNotWidenAFrameThatFits(t *testing.T) {
	frame := strings.Join([]string{strings.Repeat("x", 40), strings.Repeat(" ", 40)}, "\n")

	out := overlayNotice(frame, "short", 40, 2)

	for i, ln := range strings.Split(stripANSI(out), "\n") {
		if got := lipgloss.Width(ln); got > 40 {
			t.Errorf("row %d is %d cells wide in a 40-column terminal", i, got)
		}
	}
}

// TestView_NoticeIsVisibleAtTheSmallestSupportedSize exercises the real frame
// rather than a synthetic one: the notice must reach the user at 80×24, the
// size the whole layout is budgeted against.
func TestView_NoticeIsVisibleAtTheSmallestSupportedSize(t *testing.T) {
	m := sm_sendSize(sandboxedModel(t), 80, 24).(Model)
	m.persistErr = "Couldn't save result to disk"

	row := noticeRow(m.View().Content, "Couldn't save result to disk")
	if row < 0 {
		t.Fatal("the notice never reached the frame")
	}
	if row > 23 {
		t.Errorf("notice on row %d of a 24-row terminal — the user never sees it", row)
	}
}
