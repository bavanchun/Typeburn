package ui

import (
	"fmt"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// settingRow is one row in the settings list: a label, a list of possible
// string values, the current index, and a contextual help string.
type settingRow struct {
	label  string
	values []string
	idx    int
	help   string
}

// row indices — fixed, never reordered.
const (
	rowTheme         = 0
	rowDefaultMode   = 1
	rowDefaultLength = 2
	rowBlinkCursor   = 3
	rowStrictMode    = 4
	rowPunctuation   = 5
	rowNumbers       = 6
)

var selectableDefaultModes = []string{"time", "words", "quote"}

// buildRows constructs the 7 fixed settings rows from the current settings pointer.
func buildRows(s *config.Settings) []settingRow {
	// Theme row: cycles the available themes.
	themeVals := theme.Available()
	themeIdx := 0
	for i, v := range themeVals {
		if v == s.Theme {
			themeIdx = i
			break
		}
	}

	// Default mode row.
	modeVals := append([]string(nil), selectableDefaultModes...)
	modeIdx := 0
	for i, v := range modeVals {
		if string(s.DefaultMode) == v {
			modeIdx = i
			break
		}
	}
	if s.DefaultMode == config.ModeCode {
		modeVals = append(modeVals, "code")
		modeIdx = len(modeVals) - 1
	}

	// Default length row: option list depends on current default mode.
	lenVals, lenIdx := buildLengthRow(s.DefaultMode, s.DefaultLength)

	// Blink cursor row.
	blinkVals := []string{"off", "on"}
	blinkIdx := 0
	if s.BlinkCursor {
		blinkIdx = 1
	}

	// Strict mode row.
	strictVals := []string{"off", "on"}
	strictIdx := 0
	if s.StrictMode {
		strictIdx = 1
	}

	// Punctuation row.
	punctuationVals := []string{"off", "on"}
	punctuationIdx := 0
	if s.Punctuation {
		punctuationIdx = 1
	}

	// Numbers row.
	numbersVals := []string{"off", "on"}
	numbersIdx := 0
	if s.Numbers {
		numbersIdx = 1
	}

	return []settingRow{
		{
			label:  "Theme",
			values: themeVals,
			idx:    themeIdx,
			help:   "Color scheme applied across all screens.",
		},
		{
			label:  "Default mode",
			values: modeVals,
			idx:    modeIdx,
			help:   "Mode pre-selected on the Home screen at startup.",
		},
		{
			label:  "Default length",
			values: lenVals,
			idx:    lenIdx,
			help:   "Length option pre-selected for the default mode.",
		},
		{
			label:  "Blink cursor",
			values: blinkVals,
			idx:    blinkIdx,
			help:   "Toggle cursor blink (530 ms) during the typing test.",
		},
		{
			label:  "Strict mode",
			values: strictVals,
			idx:    strictIdx,
			help:   "Block wrong keys: cursor will not advance past an error.",
		},
		{
			label:  "Punctuation",
			values: punctuationVals,
			idx:    punctuationIdx,
			help:   "Add commas, periods, and capitalization to Words/Time tests.",
		},
		{
			label:  "Numbers",
			values: numbersVals,
			idx:    numbersIdx,
			help:   "Mix in random numbers for Words/Time tests.",
		},
	}
}

// buildLengthRow returns the string option list and the matching index for
// defaultLength within the given mode. Quote mode exposes the bucket labels
// (short/medium/long) defined on the Home screen; Code has no length option;
// numeric modes use their LengthsFor option set.
func buildLengthRow(mode config.Mode, defaultLength int) ([]string, int) {
	if mode == config.ModeQuote {
		return quoteBucketLabels, 1 // default to "medium"
	}
	if mode == config.ModeCode {
		return []string{"n/a"}, 0
	}
	lens := config.LengthsFor(mode)
	vals := make([]string, len(lens))
	idx := len(lens) / 2 // fallback: middle option
	for i, v := range lens {
		vals[i] = fmt.Sprintf("%d", v)
		if v == defaultLength {
			idx = i
		}
	}
	return vals, idx
}
