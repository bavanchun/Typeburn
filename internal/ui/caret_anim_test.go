package ui

import (
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/typing"
)

// statesFromTyped builds a states slice: n0 leading Correct cells, a Current
// cursor cell, then the rest Untyped — the common "typed a prefix" shape.
func statesTyped(correct, total int) []typing.CharState {
	st := make([]typing.CharState, total)
	for i := range st {
		switch {
		case i < correct:
			st[i] = typing.Correct
		case i == correct:
			st[i] = typing.Current
		default:
			st[i] = typing.Untyped
		}
	}
	return st
}

func TestBlinkOnAt_Phase(t *testing.T) {
	if !blinkOnAt(0) {
		t.Error("blinkOnAt(0) should be on (pre-start steady)")
	}
	if !blinkOnAt(10) { // 10/265 = 0 → on
		t.Error("blinkOnAt(10) should be on")
	}
	if blinkOnAt(300) { // 300/265 = 1 → off
		t.Error("blinkOnAt(300) should be off")
	}
	if !blinkOnAt(600) { // 600/265 = 2 → on
		t.Error("blinkOnAt(600) should be on")
	}
}

// A disabled caret must render byte-identically to the static stream.
func TestRenderWordStreamAnim_DisabledMatchesStatic(t *testing.T) {
	th := theme.Default()
	target := runesOf("hello world")
	typed := runesOf("hello")
	states := statesTyped(5, len(target))

	static := RenderWordStream(states, target, typed, 40, th)
	anim := renderWordStreamAnim(states, target, typed, 40, th, disabledCaret(), &streamTokenCache{})
	if static != anim {
		t.Errorf("disabled caret differs from static render:\nstatic=%q\nanim  =%q", static, anim)
	}
}

// A settled caret (fade window elapsed, blink on) applies no overrides, so it
// also matches the static render exactly — the end frame is byte-identical.
func TestRenderWordStreamAnim_SettledMatchesStatic(t *testing.T) {
	th := theme.Default()
	target := runesOf("hello world")
	typed := runesOf("hello")
	states := statesTyped(5, len(target))

	ca := caretAnim{nowMs: 100000, lastKeyMs: 100000 - 1000, blinkOn: true, cursorIdx: 5}
	if ca.fadeActive() {
		t.Fatal("precondition: fade should be inactive after 1s")
	}
	static := RenderWordStream(states, target, typed, 40, th)
	anim := renderWordStreamAnim(states, target, typed, 40, th, ca, &streamTokenCache{})
	if static != anim {
		t.Errorf("settled caret differs from static render")
	}
}

// During the fade window the freshly-typed cell is restyled, so the animated
// output differs from static — but only in SGR, never in runes/width.
func TestRenderWordStreamAnim_FadeChangesStyleOnly(t *testing.T) {
	th := theme.Default()
	target := runesOf("hello world")
	typed := runesOf("hello")
	states := statesTyped(5, len(target))

	ca := caretAnim{nowMs: 100000, lastKeyMs: 100000, blinkOn: true, cursorIdx: 5}
	static := RenderWordStream(states, target, typed, 40, th)
	anim := renderWordStreamAnim(states, target, typed, 40, th, ca, &streamTokenCache{})

	if static == anim {
		t.Error("fade-active caret should differ from static (new-cell fade not applied)")
	}
	if strip(static) != strip(anim) {
		t.Errorf("fade changed runes/layout, not just SGR:\nstatic=%q\nanim=%q", strip(static), strip(anim))
	}
}

// Under NO_COLOR the caret animation must stay layout-identical: same runes,
// same line count, same per-line width — only attributes differ.
func TestRenderWordStreamAnim_NoColorLayoutIdentical(t *testing.T) {
	th := theme.Load("default", true) // NO_COLOR / attribute-only
	target := runesOf("hello world")
	typed := runesOf("hello")
	states := statesTyped(5, len(target))

	static := RenderWordStream(states, target, typed, 40, th)
	// blink off + fade active = the most overrides under NO_COLOR.
	ca := caretAnim{nowMs: 100000, lastKeyMs: 100000, blinkOn: false, cursorIdx: 5}
	anim := renderWordStreamAnim(states, target, typed, 40, th, ca, &streamTokenCache{})

	if strip(static) != strip(anim) {
		t.Errorf("NO_COLOR caret not layout-identical:\nstatic=%q\nanim=%q", strip(static), strip(anim))
	}
}

// caretCellStyle gating: cursor blinks; fade/trail only apply to Correct cells
// inside the window.
func TestCaretCellStyle_Gating(t *testing.T) {
	th := theme.Default()
	on := caretAnim{nowMs: 1000, lastKeyMs: 1000, blinkOn: true, cursorIdx: 3}
	off := caretAnim{nowMs: 1000, lastKeyMs: 1000, blinkOn: false, cursorIdx: 3}

	// d=0 cursor: no override when on (base draws block), override when off.
	if _, ok := on.caretCellStyle(0, typing.Current, th); ok {
		t.Error("cursor d=0 with blink on should use base (no override)")
	}
	if _, ok := off.caretCellStyle(0, typing.Current, th); !ok {
		t.Error("cursor d=0 with blink off should override (faint)")
	}
	// d=1 new cell: override only when Correct + in window.
	if _, ok := on.caretCellStyle(1, typing.Correct, th); !ok {
		t.Error("new cell d=1 Correct in-window should override")
	}
	if _, ok := on.caretCellStyle(1, typing.Incorrect, th); ok {
		t.Error("new cell d=1 non-Correct should not override")
	}
	// Out of window → no fade override.
	stale := caretAnim{nowMs: 5000, lastKeyMs: 1000, blinkOn: true, cursorIdx: 3}
	if _, ok := stale.caretCellStyle(1, typing.Correct, th); ok {
		t.Error("new cell d=1 out of window should not override")
	}
	if _, ok := stale.caretCellStyle(2, typing.Correct, th); ok {
		t.Error("trail d=2 out of window should not override")
	}
}

// The prefix-token cache must be reused across frames with no content change,
// and rebuilt after invalidation — this is what keeps animated allocs bounded.
func TestPrefixCacheReuse(t *testing.T) {
	th := theme.Default()
	target := runesOf("hello world foo")
	typed := runesOf("hello")
	states := statesTyped(5, len(target))
	ca := caretAnim{nowMs: 100000, lastKeyMs: 100000, blinkOn: true, cursorIdx: 5}
	cache := &streamTokenCache{}

	_ = renderWordStreamAnim(states, target, typed, 40, th, ca, cache)
	if !cache.valid {
		t.Fatal("cache should be valid after first render")
	}
	firstArr := &cache.base[0]

	_ = renderWordStreamAnim(states, target, typed, 40, th, ca, cache)
	if &cache.base[0] != firstArr {
		t.Error("cache base rebuilt on a no-change frame (not reused)")
	}

	cache.invalidate()
	if cache.valid {
		t.Error("invalidate did not clear valid")
	}
	_ = renderWordStreamAnim(states, target, typed, 40, th, ca, cache)
	if !cache.valid {
		t.Error("cache not rebuilt after invalidation")
	}
}

// HasActiveAnim opens only inside the fade window after a keystroke.
func TestHasActiveAnim_Window(t *testing.T) {
	m := newTestTyping(config.ModeWords, 10)
	if m.HasActiveAnim(1000) {
		t.Error("no keystroke yet → inactive")
	}
	m.lastKeyMs = 1000
	if !m.HasActiveAnim(1000) {
		t.Error("at keystroke → active")
	}
	if !m.HasActiveAnim(1000 + caretFadeMs - 1) {
		t.Error("inside window → active")
	}
	if m.HasActiveAnim(1000 + caretFadeMs) {
		t.Error("at window end → inactive (self-stop)")
	}
}
