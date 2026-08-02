package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
	"github.com/bavanchun/Typeburn/v2/internal/words"
)

// wrapRows wraps target at width with every cell untyped and the caret at
// caretIdx, and returns the stripped rows plus the caret's row index.
func wrapRows(t *testing.T, target string, width, caretIdx int) ([]string, int) {
	t.Helper()
	th := theme.Load("default", true)
	tgt := []rune(target)
	states := statesWithCaret(len(tgt), caretIdx)
	rows, caretRow := wrapTokens(buildWordTokens(states, tgt, nil, th), states, tgt, nil, width)
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = strip(r)
	}
	return out, caretRow
}

// TestWrapTokens_BreaksOnlyBetweenWords asserts no row ends inside a word.
//
// The check is on the reassembled text rather than on the rows: every break
// offset must land on a space, so reading a row never leaves half a word
// stranded on the next one. Words longer than the row are exempt because no
// break point exists that would make them fit.
func TestWrapTokens_BreaksOnlyBetweenWords(t *testing.T) {
	m := newTypingWithSeed(config.ModeWords, 60, words.QuoteShort,
		theme.Default(), config.DefaultKeymap(), false, false, false, false, 42)

	tgt := []rune(m.target)
	for _, width := range []int{16, 20, 37, 56, 72, 98, 164} {
		rows, _ := wrapRows(t, m.target, width, -1)
		at := 0 // offset in the target of the first rune of the next row
		for i, row := range rows {
			at += len([]rune(row))
			if at >= len(tgt) {
				break
			}
			switch {
			case tgt[at] == ' ':
				at++ // the space was folded into the break: a word boundary
			case tgt[at-1] == ' ':
				// the row ends with the space: also a word boundary
			case wordWidthAt(tgt, at) > width:
				// no break point would make this word fit
			default:
				t.Errorf("width %d row %d ends mid-word: %q | %q",
					width, i, row, rows[i+1])
			}
		}
		if at != len(tgt) {
			t.Errorf("width %d: rows cover %d of %d target runes", width, at, len(tgt))
		}
	}
}

// wordWidthAt returns the cell width of the whitespace-delimited word covering
// offset.
func wordWidthAt(rs []rune, offset int) int {
	start := offset
	for start > 0 && rs[start-1] != ' ' {
		start--
	}
	end := offset
	for end < len(rs) && rs[end] != ' ' {
		end++
	}
	w := 0
	for _, r := range rs[start:end] {
		w += cellWidth(r)
	}
	return w
}

// TestWrapTokens_OverlongWordSplits asserts a word wider than the row still
// breaks. Dropping the hard split to get word-boundary wrapping would push the
// row past the terminal edge, which is the defect being fixed, not a fix.
func TestWrapTokens_OverlongWordSplits(t *testing.T) {
	rows, _ := wrapRows(t, "hi "+strings.Repeat("x", 30), 10, -1)
	if len(rows) < 4 {
		t.Fatalf("a 30-cell word at width 10 must split across rows, got %d rows: %q", len(rows), rows)
	}
	for i, row := range rows {
		if w := lipgloss.Width(row); w > 10 {
			t.Errorf("row %d is %d cells wide, width is 10: %q", i, w, row)
		}
	}
	if got := strings.Join(rows, ""); !strings.HasSuffix(got, strings.Repeat("x", 30)) {
		t.Errorf("split lost runes: %q", got)
	}
}

// TestWrapTokens_ExactFillEmitsNoBlankRow asserts a word that exactly fills a
// row does not push the following space onto a row of its own. A blank row
// costs a full line of a window that is only a handful of lines tall.
func TestWrapTokens_ExactFillEmitsNoBlankRow(t *testing.T) {
	rows, _ := wrapRows(t, "abcde fghij klmno", 5, -1)
	want := []string{"abcde", "fghij", "klmno"}
	if len(rows) != len(want) {
		t.Fatalf("want %d rows %q, got %d: %q", len(want), want, len(rows), rows)
	}
	for i := range want {
		if strings.TrimRight(rows[i], " ") != want[i] {
			t.Errorf("row %d: want %q, got %q", i, want[i], rows[i])
		}
	}
}

// TestWrapTokens_CaretOnFoldedSpaceStaysVisible asserts the one space that is
// not folded away: when the caret sits on it, it draws a cursor block, so
// dropping it would hide the key the user has to press next.
func TestWrapTokens_CaretOnFoldedSpaceStaysVisible(t *testing.T) {
	const target = "abcde fghij"
	rows, caretRow := wrapRows(t, target, 5, 5) // index 5 is the space
	if caretRow < 0 || caretRow >= len(rows) {
		t.Fatalf("caret row %d out of range for %q", caretRow, rows)
	}
	if got := rows[caretRow]; got != "abcde " {
		t.Errorf("caret space must hang at the end of its row, got %q in %q", got, rows)
	}
}

// TestWrapTokens_MeasuresCellsNotRunes asserts wide runes are accounted at
// their real column count. Counting runes here is what let a CJK stream draw
// roughly twice the width it was given.
func TestWrapTokens_MeasuresCellsNotRunes(t *testing.T) {
	const width = 20
	for _, target := range []string{
		strings.Repeat("函数 ", 12),
		strings.Repeat("🚀 ", 12),
		"函数 returns 一个 value 🚀 done 我们 need more 宽 字 符",
	} {
		rows, _ := wrapRows(t, target, width, -1)
		for i, row := range rows {
			if w := lipgloss.Width(row); w > width {
				t.Errorf("%q row %d is %d cells wide, width is %d: %q", target, i, w, width, row)
			}
		}
	}
}

// TestWrapTokens_ZeroWidthRunesTakeNoCell asserts a rune that draws nothing
// costs nothing. A control character that consumed a layout cell shortened
// every row it appeared in, and a caret landing on one was invisible.
func TestWrapTokens_ZeroWidthRunesTakeNoCell(t *testing.T) {
	plain, _ := wrapRows(t, "abcd efgh ijkl mnop", 9, -1)
	// A combining acute, a zero-width space and a control character: three
	// runes the terminal advances the cursor by zero columns for.
	marked, _ := wrapRows(t, "a\u0301bcd e\u200bfgh i\x01jkl mnop", 9, -1)

	if len(plain) != len(marked) {
		t.Fatalf("zero-width runes changed the row count: %d vs %d (%q vs %q)",
			len(plain), len(marked), plain, marked)
	}
	for i := range plain {
		if got, want := lipgloss.Width(marked[i]), lipgloss.Width(plain[i]); got != want {
			t.Errorf("row %d: %d cells with zero-width runes, %d without", i, got, want)
		}
	}
}

// TestWrapTokens_CaretRowMatchesTheRowHoldingTheCursor asserts the reported
// caret row is the row the cursor block is actually rendered in — the index
// the viewport scrolls by. A row-count-based guess would be right at the start
// of a run and wrong everywhere the wrapping folds a space.
func TestWrapTokens_CaretRowMatchesTheRowHoldingTheCursor(t *testing.T) {
	th := theme.Load("default", false)
	m := newTypingWithSeed(config.ModeWords, 40, words.QuoteShort,
		th, config.DefaultKeymap(), false, false, false, false, 42)
	tgt := []rune(m.target)
	cursor := th.Style(theme.RoleCursorBg)

	for idx := 0; idx < len(tgt); idx++ {
		states := statesWithCaret(len(tgt), idx)
		rows, caretRow := wrapTokens(buildWordTokens(states, tgt, nil, th), states, tgt, nil, 30)
		if caretRow < 0 || caretRow >= len(rows) {
			t.Fatalf("caret index %d: row %d is outside %d rows", idx, caretRow, len(rows))
		}
		want := cursor.Render(string(tgt[idx]))
		if !strings.Contains(rows[caretRow], want) {
			t.Fatalf("caret index %d: row %d does not hold the cursor cell", idx, caretRow)
		}
	}
}

// TestWrapTokens_NoCursorReportsNoRow asserts a finished run reports -1 rather
// than a row that happens to be zero: the viewport treats -1 as "show the end".
func TestWrapTokens_NoCursorReportsNoRow(t *testing.T) {
	tgt := []rune("all done here")
	states := make([]typing.CharState, len(tgt))
	for i := range states {
		states[i] = typing.Correct
	}
	_, caretRow := wrapTokens(
		buildWordTokens(states, tgt, tgt, theme.Load("default", true)),
		states, tgt, tgt, 8)
	if caretRow != -1 {
		t.Errorf("want -1 with no cursor cell, got %d", caretRow)
	}
}
