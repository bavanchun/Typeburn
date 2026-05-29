package app

// Screen enumerates the top-level screens. The root model's Update switches on
// this value to route messages to the correct sub-model.
type Screen int

const (
	ScreenHome Screen = iota
	ScreenTyping
	ScreenResult
	ScreenSettings
	ScreenHistory
	ScreenCodePaste
)
