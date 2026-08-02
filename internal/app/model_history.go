package app

import (
	"math"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/storage"
	"github.com/bavanchun/Typeburn/v2/internal/ui"
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
	// Code mode stores the rune count of the snippet as display-only length.
	if msg.Mode == config.ModeCode {
		length = len([]rune(msg.CodeText))
	}
	accuracy := msg.Result.Accuracy
	if msg.Strict {
		accuracy = msg.Result.KeystrokeAccuracy
	}
	return storage.Record{
		Time:        nowUTC(),
		Mode:        mode,
		Length:      length,
		WPM:         int(math.Round(msg.Result.NetWPM)),
		NetWPM:      msg.Result.NetWPM,
		RawWPM:      msg.Result.RawWPM,
		Accuracy:    accuracy,
		Consistency: msg.Result.Consistency,
		Strict:      msg.Strict,
	}
}

// afkNotice tells the user why a run they finished is missing from history.
// Dropping it silently would look like the app lost the run, which is a worse
// failure than the impossible score the withholding exists to keep out.
const afkNotice = "paused mid-test — not saved to history"

// resultOutcome is what a completed run does to stored history: whether it is
// written, whether it ranks, and what the user is told. Deciding it separately
// from applying it keeps the rule assertable without a disk write standing in
// for it.
type resultOutcome struct {
	record storage.Record
	store  bool
	isBest bool
	notice string
}

// decideOutcome resolves a finished run against the history already on disk.
//
// A run whose clock was cut back by trailing inactivity is shown but never
// written, and never ranked. Its rates describe the burst the user typed before
// they walked away, not a test they took, so a stored record of it would stand
// as a personal best nobody could beat by typing.
func decideOutcome(msg ui.ResultMsg, hist []storage.Record) resultOutcome {
	if msg.Result.AFKTrimmed {
		return resultOutcome{notice: afkNotice}
	}
	rec := buildRecord(msg)
	return resultOutcome{record: rec, store: true, isBest: storage.IsNewBest(hist, rec)}
}

// handleResultMsg processes a completed test: persists the record, detects
// new-best, and builds the ResultModel with isBest set appropriately.
// It mutates the model in place and returns it ready for ScreenResult.
//
// This is the only place results are persisted, so it is the only place the
// eligibility rules have to hold.
func (m Model) handleResultMsg(msg ui.ResultMsg) Model {
	out := decideOutcome(msg, storage.LoadHistory())
	if out.notice != "" {
		m.persistErr = out.notice
	}
	if out.store {
		// Persist regardless of the new-best result. A write failure is non-fatal
		// to the session but must not be silent — surface a dismissible notice.
		//
		// A notice without an error means the record reached disk but something
		// about how it got there is worth saying: the previous file was corrupt
		// and has been set aside, or the write went ahead without the
		// cross-process lock. Both leave the result saved, so neither is an
		// error, and both are things the user would rather know than not.
		_, notice, err := storage.AppendHistoryWithNotice(out.record)
		switch {
		case err != nil:
			m.persistErr = "Couldn't save result to disk"
		case !notice.IsZero():
			m.persistErr = notice.Message
		}
	}

	m.result = ui.NewResult(msg, m.theme, m.keys).
		WithBest(out.isBest).
		WithUpdateHint(m.updateHint).
		WithRevealStart(nowUTC().UnixMilli()).
		SetSize(m.w, m.h)
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
