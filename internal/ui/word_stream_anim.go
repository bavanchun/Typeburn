package ui

import (
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// streamTokenCache holds the static prefix of styled word-stream tokens so the
// caret animation only re-Renders the ≤3 animated cells per frame instead of
// every rune. The v2 cell-diff renderer saves output bytes but NOT the upstream
// per-rune Style.Render/alloc cost, which would otherwise run for the whole
// stream at 33fps during the fade window. The cache is invalidated on any
// content or theme change (keystroke/backspace/restart/new-test/theme swap);
// width is irrelevant because wrapping is a separate pass over the same tokens.
type streamTokenCache struct {
	base  []string
	valid bool
}

// invalidate marks the cache stale; the next render rebuilds the base tokens.
func (c *streamTokenCache) invalidate() {
	if c != nil {
		c.valid = false
	}
}

// renderWordStreamAnim renders the word-stream with caret animation applied. It
// reuses cached base tokens (rebuilding only on content/theme change) and
// re-Renders only the ≤3 animated cells around the cursor, then wraps. With a
// disabled caret (cursorIdx < 0) the output equals RenderWordStream exactly.
func renderWordStreamAnim(
	states []typing.CharState,
	target []rune,
	typed []rune,
	width int,
	th theme.Theme,
	ca caretAnim,
	cache *streamTokenCache,
) string {
	if width < 1 {
		width = 40
	}

	var base []string
	if cache != nil && cache.valid && len(cache.base) == len(states) {
		base = cache.base
	} else {
		base = buildWordTokens(states, target, typed, th)
		if cache != nil {
			cache.base = base
			cache.valid = true
		}
	}

	tokens := base
	if ca.cursorIdx >= 0 {
		tokens = make([]string, len(base))
		copy(tokens, base)
		applyCaretOverrides(tokens, states, target, typed, ca, th)
	}
	return wrapTokens(tokens, states, target, typed, width)
}

// applyCaretOverrides re-Renders the cursor cell and the ≤2 cells behind it with
// their animated styles. It never changes a cell's rune or width — only its SGR.
func applyCaretOverrides(
	tokens []string,
	states []typing.CharState,
	target, typed []rune,
	ca caretAnim,
	th theme.Theme,
) {
	for d := 0; d <= 2; d++ {
		i := ca.cursorIdx - d
		if i < 0 || i >= len(tokens) {
			continue
		}
		st, ok := ca.caretCellStyle(d, states[i], th)
		if !ok {
			continue
		}
		tokens[i] = st.Render(string(runeAtIndex(i, target, typed)))
	}
}
