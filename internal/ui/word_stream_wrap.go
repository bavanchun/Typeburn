package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// cellWidth returns how many terminal columns one rune occupies.
//
// Counting runes instead of cells is what lets a wide-rune line run off the
// screen: a CJK glyph is one rune but two columns, and a combining mark or a
// control character is one rune but none. ASCII printables are answered
// directly because this is asked once per rune per frame; everything else goes
// through lipgloss, which is the same measurement the frame-fit assertions use.
func cellWidth(r rune) int {
	if r >= 0x20 && r < 0x7f {
		return 1
	}
	return lipgloss.Width(string(r))
}

// wrapCell is one styled rune plus the two facts the wrapper needs about it:
// how wide it is, and whether the caret sits on it.
type wrapCell struct {
	tok   string
	w     int
	caret bool
}

// wordWrapper assembles styled rune tokens into rows, holding back a run of
// non-space runes until its full width is known. That one-word lookahead is
// what keeps a line from ending in the middle of a word — a break mid-word
// destroys the read-ahead a typing test depends on.
type wordWrapper struct {
	width    int
	rows     []string
	line     strings.Builder
	lineW    int
	caretRow int

	word  []wrapCell // pending run of non-space runes
	wordW int
}

// breakLine ends the current row and starts an empty one.
func (ww *wordWrapper) breakLine() {
	ww.rows = append(ww.rows, ww.line.String())
	ww.line.Reset()
	ww.lineW = 0
}

// putCell writes one cell, breaking first when it no longer fits. A rune wider
// than an entire row is still written: a rune cannot be split, so the row is
// allowed to be one rune wide instead of the loop never terminating.
func (ww *wordWrapper) putCell(c wrapCell) {
	if ww.lineW > 0 && ww.lineW+c.w > ww.width {
		ww.breakLine()
	}
	if c.caret {
		ww.caretRow = len(ww.rows) // the caret lands in the row being built now
	}
	ww.line.WriteString(c.tok)
	ww.lineW += c.w
}

// putSpace writes the space between two words.
//
// A space that no longer fits the row is folded into the break rather than
// starting the next row with a blank column: an untyped or correctly typed
// space draws nothing, so at a row end it is indistinguishable from the row
// simply ending. A highlighted space — the caret is on it, or it was mistyped —
// does draw, so it hangs at the end of its own row instead of vanishing; the
// user still has to press that key.
func (ww *wordWrapper) putSpace(c wrapCell, drawn bool) {
	if ww.lineW == 0 || ww.lineW+c.w <= ww.width {
		ww.putCell(c)
		return
	}
	if drawn {
		if c.caret {
			ww.caretRow = len(ww.rows)
		}
		ww.line.WriteString(c.tok)
		ww.lineW += c.w
	}
	ww.breakLine()
}

// spaceDrawsAGlyph reports whether a space cell renders something the user can
// see: the cursor block, or an error highlight. Correct and untyped spaces are
// blank either way.
func spaceDrawsAGlyph(s typing.CharState) bool {
	switch s {
	case typing.Current, typing.Incorrect, typing.IncorrectSpace, typing.Extra:
		return true
	default:
		return false
	}
}

// flushWord places the pending word. A word that fits a row but not the rest of
// this one moves down whole; a word wider than a whole row falls through to
// putCell and hard-splits, because no break point would make it fit.
func (ww *wordWrapper) flushWord() {
	if len(ww.word) == 0 {
		return
	}
	if ww.wordW <= ww.width && ww.lineW > 0 && ww.lineW+ww.wordW > ww.width {
		ww.breakLine()
	}
	for _, c := range ww.word {
		ww.putCell(c)
	}
	ww.word, ww.wordW = ww.word[:0], 0
}

// wrapTokens assembles the styled rune tokens into rows that fit width, wrapping
// on word boundaries and measuring in terminal cells rather than runes.
//
// It returns the rows and the index of the row holding the caret (-1 when no
// cell is Current), so a caller can window the rows around the caret.
func wrapTokens(
	tokens []string,
	states []typing.CharState,
	target []rune,
	typed []rune,
	width int,
) ([]string, int) {
	if width < 1 {
		width = 1
	}
	ww := wordWrapper{width: width, caretRow: -1}

	for i, tok := range tokens {
		var st typing.CharState
		if i < len(states) {
			st = states[i]
		}
		r := runeAtIndex(i, target, typed)
		c := wrapCell{tok: tok, w: cellWidth(r), caret: st == typing.Current}
		if r != ' ' {
			ww.word = append(ww.word, c)
			ww.wordW += c.w
			continue
		}
		ww.flushWord()
		ww.putSpace(c, spaceDrawsAGlyph(st))
	}
	ww.flushWord()

	if ww.line.Len() > 0 {
		ww.rows = append(ww.rows, ww.line.String())
	}
	return ww.rows, ww.caretRow
}
