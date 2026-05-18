package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// SettingsModel is the sub-model for the Settings screen. It owns exactly
// 4 rows (Theme, Default mode, Default length, Blink cursor) and nothing else.
// Every value change calls onChange so the root can persist and apply live.
// Row types and row-building helpers live in settings_rows.go.
type SettingsModel struct {
	rows     []settingRow
	sel      int // currently-selected row index (0-3)
	s        *config.Settings
	th       theme.Theme
	km       config.Keymap
	w, h     int
	onChange func(config.Settings)
}

// NewSettings constructs the SettingsModel wired to the root settings pointer.
// onChange is called with the updated Settings on every value change so the
// root can persist atomically and propagate live (theme rebuild, blink update).
func NewSettings(s *config.Settings, th theme.Theme, km config.Keymap, onChange func(config.Settings)) SettingsModel {
	m := SettingsModel{s: s, th: th, km: km, onChange: onChange}
	m.rows = buildRows(s)
	return m
}

// SetSize stores terminal dimensions for layout.
func (m SettingsModel) SetSize(w, h int) SettingsModel {
	m.w, m.h = w, h
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

	case m.km.OptRight.Matches(k), m.km.Cycle.Matches(k):
		m = m.cycleSelected(+1)

	case m.km.Back.Matches(k), m.km.NavHome.Matches(k):
		// Settings already persisted on each change; esc just returns to Home.
		return m, func() tea.Msg { return AbortMsg{} }
	}

	return m, nil
}

// cycleSelected advances the selected row's value by delta (+1 or -1) with wrap.
// It applies the change to the root settings pointer and calls onChange.
func (m SettingsModel) cycleSelected(delta int) SettingsModel {
	row := &m.rows[m.sel]
	n := len(row.values)
	row.idx = ((row.idx+delta)%n + n) % n

	// Apply the new value to the settings struct and propagate.
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

	if m.onChange != nil {
		m.onChange(*m.s)
	}
	return m
}

// applyRow writes the selected value back into the settings pointer.
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
	}
}
