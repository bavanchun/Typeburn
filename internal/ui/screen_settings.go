package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// SettingsModel is the sub-model for the Settings screen. It owns exactly
// 7 rows (Theme, Default mode, Default length, Blink cursor, Strict mode,
// Punctuation, Numbers) and nothing else.
// It holds its settings BY VALUE; every value change emits a
// SettingsChangedMsg so the root can persist and apply it to the live model.
// (A callback/pointer bound in app.New() would target a copied-out struct the
// program never renders.) Row types/helpers live in settings_rows.go.
type SettingsModel struct {
	rows []settingRow
	sel  int // currently-selected row index (0-6)
	s    config.Settings
	th   theme.Theme
	km   config.Keymap
	w, h int
}

// NewSettings constructs the SettingsModel from a settings value.
func NewSettings(s config.Settings, th theme.Theme, km config.Keymap) SettingsModel {
	m := SettingsModel{s: s, th: th, km: km}
	m.rows = buildRows(&m.s)
	return m
}

// SetSize stores terminal dimensions for layout.
func (m SettingsModel) SetSize(w, h int) SettingsModel {
	m.w, m.h = w, h
	return m
}

// Sel returns the selected row index (so the root can preserve it on rebuild).
func (m SettingsModel) Sel() int { return m.sel }

// WithSel restores a previously-selected row index, clamped to the row count.
func (m SettingsModel) WithSel(sel int) SettingsModel {
	if sel >= 0 && sel < len(m.rows) {
		m.sel = sel
	}
	return m
}

// Update handles key events for the Settings screen.
func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	k := kp.Key()

	switch {
	case m.km.Up.Matches(k):
		if m.sel > 0 {
			m.sel--
		}

	case m.km.Down.Matches(k):
		if m.sel < len(m.rows)-1 {
			m.sel++
		}

	case m.km.OptLeft.Matches(k):
		m = m.cycleSelected(-1)
		return m, m.changedCmd()

	case m.km.OptRight.Matches(k), m.km.Cycle.Matches(k):
		m = m.cycleSelected(+1)
		return m, m.changedCmd()

	case m.km.Back.Matches(k), m.km.NavHome.Matches(k):
		// Settings already applied on each change; esc just returns to Home.
		return m, func() tea.Msg { return AbortMsg{} }
	}

	return m, nil
}

// changedCmd emits the current settings value to the root for live apply.
func (m SettingsModel) changedCmd() tea.Cmd {
	s := m.s
	return func() tea.Msg { return SettingsChangedMsg{Settings: s} }
}

// cycleSelected advances the selected row's value by delta (+1 or -1) with wrap
// and writes it into the local settings value.
func (m SettingsModel) cycleSelected(delta int) SettingsModel {
	row := &m.rows[m.sel]
	n := len(row.values)
	row.idx = ((row.idx+delta)%n + n) % n

	m.applyRow(m.sel, row.idx)

	// When default mode changes, rebuild the length row to match the new mode.
	if m.sel == rowDefaultMode {
		newMode := config.Mode(m.rows[rowDefaultMode].values[m.rows[rowDefaultMode].idx])
		lenVals, lenIdx := buildLengthRow(newMode, m.s.DefaultLength)
		m.rows[rowDefaultLength].values = lenVals
		m.rows[rowDefaultLength].idx = lenIdx
		// Also apply clamped length back to settings.
		m.applyRow(rowDefaultLength, lenIdx)
	}

	return m
}

// applyRow writes the selected value into the local settings value.
func (m *SettingsModel) applyRow(rowIdx, valIdx int) {
	val := m.rows[rowIdx].values[valIdx]
	switch rowIdx {
	case rowTheme:
		m.s.Theme = val
	case rowDefaultMode:
		m.s.DefaultMode = config.Mode(val)
	case rowDefaultLength:
		if m.s.DefaultMode == config.ModeQuote {
			// Quote uses bucket labels ("short"/"medium"/"long"); no int to store.
			m.s.DefaultLength = 0
		} else {
			lens := config.LengthsFor(m.s.DefaultMode)
			if valIdx < len(lens) {
				m.s.DefaultLength = lens[valIdx]
			}
		}
	case rowBlinkCursor:
		m.s.BlinkCursor = (val == "on")
	case rowStrictMode:
		m.s.StrictMode = (val == "on")
	case rowPunctuation:
		m.s.Punctuation = (val == "on")
	case rowNumbers:
		m.s.Numbers = (val == "on")
	}
}
