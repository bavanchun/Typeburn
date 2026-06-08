// Package mode defines typing-test modes and their selectable lengths.
package mode

// Mode identifies a test mode. Stored as a string for forward-compatible JSON.
type Mode string

const (
	ModeTime  Mode = "time"
	ModeWords Mode = "words"
	ModeQuote Mode = "quote"
	// ModeCode types a user-supplied snippet verbatim. It has no numeric length
	// and completes on an exact full-text match.
	ModeCode Mode = "code"
)

// LengthsFor returns the selectable length options for a mode. Quote and Code
// modes have no numeric length selector, so they return nil.
func LengthsFor(m Mode) []int {
	switch m {
	case ModeWords:
		return []int{10, 25, 50, 100}
	case ModeQuote, ModeCode:
		return nil
	default:
		return []int{15, 30, 60, 120}
	}
}
