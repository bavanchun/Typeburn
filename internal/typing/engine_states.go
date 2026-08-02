package typing

// States returns a CharState slice covering all target positions plus any extra
// typed runes. The current cursor position is marked Current; positions behind
// it are Correct/Incorrect/IncorrectSpace; positions ahead are Untyped.
//
// IncorrectSpace is assigned when the target rune is a space and the typed rune
// is not (or vice versa), making word-boundary errors visually distinct.
func (e *Engine) States() []CharState {
	cursor := len(e.typed)
	total := len(e.target)
	if cursor > total {
		total = cursor
	}

	states := make([]CharState, total)

	for i := 0; i < total; i++ {
		switch {
		case i == cursor:
			states[i] = Current
		case i > cursor:
			states[i] = Untyped
		case i >= len(e.target):
			// typed past end of target → Extra
			states[i] = Extra
		default:
			typed := e.typed[i]
			tgt := e.target[i]
			if typed == tgt {
				states[i] = Correct
			} else if tgt == ' ' || typed == ' ' {
				// wrong char at a space boundary — distinct visual class
				states[i] = IncorrectSpace
			} else {
				states[i] = Incorrect
			}
		}
	}

	return states
}
