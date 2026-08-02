package typing

// ReplayBuffer reconstructs the typed buffer an Engine holds after applying the
// whole log, and is the single definition of what a log means:
//
//   - a blocked keystroke never entered the buffer, so it is skipped. Counting
//     it would let a later backspace pop a slot the engine never had, and every
//     position after that would describe a run nobody typed;
//   - a backspace marker (Typed == 0) pops the last slot;
//   - anything else appends.
//
// The returned keystrokes are the surviving ones, in buffer order, so callers
// that need the target and correctness of each slot get them without a second
// walk of the log. Callers that need only the runes use TypedFromLog.
//
// Every consumer replays through here. Two independent copies of this loop is
// how a single strict-mode desync became two.
func ReplayBuffer(log []Keystroke) []Keystroke {
	buf := make([]Keystroke, 0, len(log))
	for _, k := range log {
		switch {
		case k.Blocked:
			continue
		case k.Typed == 0:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
		default:
			buf = append(buf, k)
		}
	}
	return buf
}

// TypedFromLog returns the rune buffer an Engine holds after applying the log,
// equivalent to Engine.Typed() for the engine that produced it.
func TypedFromLog(log []Keystroke) []rune {
	buf := ReplayBuffer(log)
	out := make([]rune, len(buf))
	for i, k := range buf {
		out[i] = k.Typed
	}
	return out
}
