// Package storage handles reading and writing user data to disk.
// history_record.go defines the persisted record type for completed typing tests.
package storage

import "time"

// Record captures all fields needed for the History screen table, trend sparkline,
// and new-best detection. It is encoded as JSON in history.json.
//
// Mode is one of "time", "words", or "quote" (mirrors config.Mode string values).
// Length is the numeric parameter for time/words modes; 0 for quote mode.
// WPM is math.Round(NetWPM) stored as int for compact display and comparison.
type Record struct {
	// Time is the wall-clock moment the test completed (RFC3339 in JSON).
	Time time.Time `json:"time"`

	// Mode is the test mode string: "time", "words", or "quote".
	Mode string `json:"mode"`

	// Length is the mode parameter (seconds for time, word count for words). 0 for quote.
	Length int `json:"length"`

	// WPM is the rounded net WPM (NetWPM rounded to nearest integer).
	WPM int `json:"wpm"`

	// RawWPM is the raw WPM (all typed chars / 5 / minutes).
	RawWPM float64 `json:"raw_wpm"`

	// Accuracy is the final accuracy percentage (0-100).
	Accuracy float64 `json:"accuracy"`

	// Consistency is the consistency percentage (0-100).
	Consistency float64 `json:"consistency"`
}
