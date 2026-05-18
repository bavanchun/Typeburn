package app

import (
	"math"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/storage"
	"github.com/bavanchun/Typeburn/internal/ui"
)

// buildRecord converts a ResultMsg into a storage.Record for persistence.
// The timestamp is set to the current wall clock time at the moment of the call.
func buildRecord(msg ui.ResultMsg) storage.Record {
	mode := string(msg.Mode)
	length := msg.Length
	// Quote mode has no numeric length; store 0 to match IsNewBest scoping.
	if msg.Mode == config.ModeQuote {
		length = 0
	}
	return storage.Record{
		Time:        nowUTC(),
		Mode:        mode,
		Length:      length,
		WPM:         int(math.Round(msg.Result.NetWPM)),
		RawWPM:      msg.Result.RawWPM,
		Accuracy:    msg.Result.Accuracy,
		Consistency: msg.Result.Consistency,
	}
}

// handleResultMsg processes a completed test: persists the record, detects
// new-best, and builds the ResultModel with isBest set appropriately.
// It mutates the model in place and returns it ready for ScreenResult.
func (m Model) handleResultMsg(msg ui.ResultMsg) Model {
	rec := buildRecord(msg)
	hist := storage.LoadHistory()
	isBest := storage.IsNewBest(hist, rec)
	// Persist regardless of IsNewBest result; ignore write errors (non-fatal).
	_, _ = storage.AppendHistory(rec)

	m.result = ui.NewResult(msg, m.theme, m.keys).WithBest(isBest).SetSize(m.w, m.h)
	m.screen = ScreenResult
	return m
}

// handleNavHistory initialises a fresh HistoryModel from disk and switches to
// the History screen. Called on NavHistoryMsg and the '3' global key.
func (m Model) handleNavHistory() Model {
	records := storage.LoadHistory()
	hist := ui.NewHistory(records, m.theme, m.keys).SetSize(m.w, m.h)
	m.hist = hist
	m.screen = ScreenHistory
	return m
}
