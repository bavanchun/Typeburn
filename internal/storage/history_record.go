// Package storage handles reading and writing user data to disk.
// history_record.go defines the persisted record type for completed typing tests.
package storage

import "time"

// Record captures all fields needed for the History screen table, trend sparkline,
// and new-best detection. It is encoded as JSON in history.json.
//
// Mode is one of "time", "words", "quote", or "code" (mirrors config.Mode).
// Length is the numeric parameter for time/words; 0 for quote; the snippet
// rune count for code (display only — code is excluded from new-best).
// WPM is math.Round(NetWPM) stored as int for compact display and comparison.
type Record struct {
	// Time is the wall-clock moment the test completed (RFC3339 in JSON).
	Time time.Time `json:"time"`

	// Mode is the test mode string: "time", "words", "quote", or "code".
	Mode string `json:"mode"`

	// Length is the mode parameter (seconds for time, word count for words). 0 for quote.
	Length int `json:"length"`

	// WPM is the rounded net WPM (NetWPM rounded to nearest integer).
	// Kept for compact display and JSON back-compat with v1.0.0 records.
	WPM int `json:"wpm"`

	// NetWPM is the unrounded net WPM used for precise new-best comparison.
	// Legacy records (written before this field existed) unmarshal as 0.0;
	// callers must fall back to float64(WPM) when NetWPM is zero.
	NetWPM float64 `json:"net_wpm"`

	// RawWPM is the raw WPM (all typed chars / 5 / minutes).
	RawWPM float64 `json:"raw_wpm"`

	// Accuracy is the final accuracy percentage (0-100).
	Accuracy float64 `json:"accuracy"`

	// Consistency is the consistency percentage (0-100).
	Consistency float64 `json:"consistency"`

	// Strict is true if the run was performed in strict (stop-on-error letter) mode.
	Strict bool `json:"strict"`
}
