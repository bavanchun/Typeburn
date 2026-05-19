package ui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/typing"
)

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func strip(s string) string { return ansiRE.ReplaceAllString(s, "") }

// allUntyped builds a states slice (all Untyped) of length n with one Current
// at caretIdx (or none if caretIdx < 0).
func statesWithCaret(n, caretIdx int) []typing.CharState {
	st := make([]typing.CharState, n)
	for i := range st {
		st[i] = typing.Untyped
	}
	if caretIdx >= 0 && caretIdx < n {
		st[caretIdx] = typing.Current
	}
	return st
}

func runesOf(s string) []rune { return []rune(s) }

func TestRenderCodeStream_LiteralNewlinesAndTabs(t *testing.T) {
	th := theme.Load("default", true) // no-color: deterministic structure

	t.Run("newline splits rows", func(t *testing.T) {
		tgt := runesOf("a\nb")
		out := strip(RenderCodeStream(statesWithCaret(len(tgt), 0), tgt, nil, 20, 10, th))
		rows := strings.Split(out, "\n")
		if len(rows) != 2 || strings.TrimRight(rows[0], " ") != "a" || strings.TrimRight(rows[1], " ") != "b" {
			t.Fatalf("want rows [a b], got %q", rows)
		}
	})

	t.Run("tab renders as 2 visual columns", func(t *testing.T) {
		tgt := runesOf("\tx")
		out := strip(RenderCodeStream(statesWithCaret(len(tgt), 0), tgt, nil, 20, 10, th))
		row0 := strings.Split(out, "\n")[0]
		if !strings.HasPrefix(row0, "  x") {
			t.Fatalf("tab should expand to 2 spaces then x; got %q", row0)
		}
	})
}

func TestRenderCodeStream_StateStylingApplied(t *testing.T) {
	th := theme.Load("default", false) // colored: states must differ visibly
	tgt := runesOf("ab")
	st := []typing.CharState{typing.Incorrect, typing.Untyped}
	out := RenderCodeStream(st, tgt, runesOf("xb"), 20, 10, th)
	// The incorrect 'a' token must carry styling distinct from a plain rune.
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI styling in colored output, got %q", out)
	}
	if !strings.Contains(strip(out), "a") || !strings.Contains(strip(out), "b") {
		t.Fatalf("plain content must survive styling, got %q", strip(out))
	}
}

func TestRenderCodeStream_Viewport(t *testing.T) {
	th := theme.Load("default", true)
	// 30 logical rows: "l0".."l29"
	var b strings.Builder
	for i := 0; i < 30; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("l")
		b.WriteByte(byte('0' + i%10))
	}
	tgt := runesOf(b.String())
	height := 10

	rowAtRune := func(idx int) string {
		out := strip(RenderCodeStream(statesWithCaret(len(tgt), idx), tgt, nil, 20, height, th))
		return out
	}

	t.Run("caret at top shows first window", func(t *testing.T) {
		rows := strings.Split(rowAtRune(0), "\n")
		if len(rows) != height {
			t.Fatalf("want %d visible rows, got %d", height, len(rows))
		}
		if strings.TrimRight(rows[0], " ") != "l0" {
			t.Fatalf("top window must start at l0, got %q", rows[0])
		}
	})

	t.Run("caret at bottom keeps last row visible", func(t *testing.T) {
		rows := strings.Split(rowAtRune(len(tgt)-1), "\n")
		if len(rows) != height {
			t.Fatalf("want %d visible rows, got %d", height, len(rows))
		}
		last := strings.TrimRight(rows[len(rows)-1], " ")
		if last != "l9" { // row 29 → "l"+(29%10)="l9"
			t.Fatalf("bottom window must end at last row l9, got %q", last)
		}
	})

	t.Run("short input not scrolled", func(t *testing.T) {
		short := runesOf("x\ny\nz")
		out := strip(RenderCodeStream(statesWithCaret(len(short), 0), short, nil, 20, height, th))
		rows := strings.Split(out, "\n")
		if len(rows) != 3 {
			t.Fatalf("short input must render exactly its rows, got %d: %q", len(rows), rows)
		}
	})
}

func TestRenderCodeStream_LongLineWraps(t *testing.T) {
	th := theme.Load("default", true)
	tgt := runesOf(strings.Repeat("a", 28)) // one logical line, no \n
	width := 10
	out := strip(RenderCodeStream(statesWithCaret(len(tgt), 27), tgt, nil, width, 10, th))
	rows := strings.Split(out, "\n")
	if len(rows) < 3 {
		t.Fatalf("28 chars at width 10 must wrap into >=3 rows, got %d", len(rows))
	}
	for _, r := range rows {
		if len(strip(r)) > width {
			t.Fatalf("row exceeds width %d: %q", width, r)
		}
	}
}
