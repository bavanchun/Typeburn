package config

import tea "charm.land/bubbletea/v2"

// chord is a single key combination: a key code plus required modifiers.
// Mod 0 means "no modifiers". Matching is exact on modifiers so that, e.g.,
// plain "c" never triggers a ctrl+c binding.
type chord struct {
	code rune
	mod  tea.KeyMod
}

// Binding is a named action with one or more accepted chords (e.g. arrows
// and vim h/l both move the option selector).
type Binding struct {
	Name   string
	chords []chord
}

// Matches reports whether a key event satisfies this binding.
func (b Binding) Matches(k tea.Key) bool {
	for _, c := range b.chords {
		if k.Code == c.code && k.Mod == c.mod {
			return true
		}
	}
	return false
}

func k(code rune) chord  { return chord{code: code} }
func ck(code rune) chord { return chord{code: code, mod: tea.ModCtrl} }
func sk(code rune) chord { return chord{code: code, mod: tea.ModShift} }
func bind(name string, cs ...chord) Binding {
	return Binding{Name: name, chords: cs}
}

// Keymap is the single source of truth for key bindings, mirroring
// docs/design-guidelines.md §8. Screens consult these bindings; they never
// hardcode keys.
type Keymap struct {
	// Global
	Quit        Binding // ctrl+c
	Back        Binding // esc
	RestartSame Binding // tab (restart same test)
	NewTest     Binding // ctrl+r (fresh test)
	NavHome     Binding // 1
	NavSettings Binding // 2
	NavHistory  Binding // 3
	// Home
	NextMode Binding // tab / shift+tab cycles; tab=next
	PrevMode Binding // shift+tab
	OptLeft  Binding // ← / h  (change length option)
	OptRight Binding // → / l
	Start    Binding // enter / space
	// Lists (settings / history / selectors)
	Up     Binding // ↑ / k
	Down   Binding // ↓ / j
	Cycle  Binding // enter (toggle/cycle selected value)
	Top    Binding // g
	Bottom Binding // G
}

// DefaultKeymap returns the v1 bindings exactly per design §8.
func DefaultKeymap() Keymap {
	return Keymap{
		Quit:        bind("quit", ck('c')),
		Back:        bind("back", k(tea.KeyEsc)),
		RestartSame: bind("restart", k(tea.KeyTab)),
		NewTest:     bind("new test", ck('r')),
		NavHome:     bind("home", k('1')),
		NavSettings: bind("settings", k('2')),
		NavHistory:  bind("history", k('3')),

		NextMode: bind("next mode", k(tea.KeyTab)),
		PrevMode: bind("prev mode", sk(tea.KeyTab)),
		OptLeft:  bind("prev option", k(tea.KeyLeft), k('h')),
		OptRight: bind("next option", k(tea.KeyRight), k('l')),
		Start:    bind("start", k(tea.KeyEnter), k(tea.KeySpace)),

		Up:     bind("up", k(tea.KeyUp), k('k')),
		Down:   bind("down", k(tea.KeyDown), k('j')),
		Cycle:  bind("cycle", k(tea.KeyEnter)),
		Top:    bind("top", k('g')),
		Bottom: bind("bottom", sk('g')),
	}
}
