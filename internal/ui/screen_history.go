package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/storage"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// visibleRows is the number of data rows shown in the history table window.
// Computed from terminal height minus fixed chrome (title, sparkline, header, meta, footer).
const historyChrome = 10 // lines consumed by non-table chrome

// HistoryModel is the sub-model for the History screen. It shows a trend
// sparkline of recent WPM and a scrollable table of completed tests.
// Rows are displayed newest-first; the underlying slice is stored oldest-first
// (matching storage order) and reversed on render.
type HistoryModel struct {
	rows []storage.Record // oldest-first (storage order)
	sel  int              // selected row index in newest-first display order
	top  int              // scroll window start index in newest-first display order
	w, h int
	th   theme.Theme
	km   config.Keymap
}

// NewHistory constructs a HistoryModel from a loaded history slice.
// records should be in storage order (oldest-first); the model reverses for display.
func NewHistory(records []storage.Record, th theme.Theme, km config.Keymap) HistoryModel {
	return HistoryModel{
		rows: records,
		th:   th,
		km:   km,
	}
}

// SetSize stores terminal dimensions. Called by the root on WindowSizeMsg.
func (m HistoryModel) SetSize(w, h int) HistoryModel {
	m.w, m.h = w, h
	return m
}

// visibleCount returns how many table rows fit in the current terminal height.
func (m HistoryModel) visibleCount() int {
	v := m.h - historyChrome
	if v < 1 {
		v = 1
	}
	return v
}

// newestFirst returns rows in newest-first display order.
func (m HistoryModel) newestFirst() []storage.Record {
	n := len(m.rows)
	out := make([]storage.Record, n)
	for i, r := range m.rows {
		out[n-1-i] = r
	}
	return out
}

// Update handles key events for the History screen per design §8.6.
func (m HistoryModel) Update(msg tea.Msg) (HistoryModel, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	k := kp.Key()

	n := len(m.rows)
	vis := m.visibleCount()

	switch {
	case m.km.Up.Matches(k):
		if m.sel > 0 {
			m.sel--
			if m.sel < m.top {
				m.top = m.sel
			}
		}

	case m.km.Down.Matches(k):
		if m.sel < n-1 {
			m.sel++
			if m.sel >= m.top+vis {
				m.top = m.sel - vis + 1
			}
		}

	case m.km.Top.Matches(k):
		m.sel = 0
		m.top = 0

	case m.km.Bottom.Matches(k):
		if n > 0 {
			m.sel = n - 1
			m.top = n - vis
			if m.top < 0 {
				m.top = 0
			}
		}

	case m.km.Back.Matches(k), m.km.NavHome.Matches(k):
		return m, func() tea.Msg { return AbortMsg{} }
	}

	return m, nil
}
