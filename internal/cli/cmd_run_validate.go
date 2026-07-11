package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/words"
)

func buildRunRequest(cmd *cobra.Command, f runFlags, settings config.Settings) (runRequest, error) {
	mode := settings.DefaultMode
	if f.mode != "" {
		mode = config.Mode(f.mode)
	}
	if !validMode(mode) {
		return runRequest{}, usageError("invalid mode %q (valid: time, words, quote, code)", f.mode)
	}
	if f.theme != "" && !containsString(theme.Names(), f.theme) {
		return runRequest{}, usageError("invalid theme %q (valid: %s)", f.theme, strings.Join(theme.Names(), ", "))
	}

	durationChanged := cmd.Flags().Changed("duration")
	wordsChanged := cmd.Flags().Changed("words")
	quoteChanged := cmd.Flags().Changed("quote-len")
	textChanged := cmd.Flags().Changed("text")
	if err := validateRunCombos(mode, f, durationChanged, wordsChanged, quoteChanged, textChanged); err != nil {
		return runRequest{}, err
	}

	length, err := runLengthForMode(mode, f, settings, durationChanged, wordsChanged)
	if err != nil {
		return runRequest{}, err
	}
	ql, err := parseQuoteLen(f.quoteLen)
	if err != nil {
		return runRequest{}, err
	}
	return runRequest{
		mode: mode, length: length, quoteLen: ql,
		theme: f.theme, textPath: f.text, noTUI: f.noTUI, json: f.json,
	}, nil
}

func validateRunCombos(mode config.Mode, f runFlags, durationChanged, wordsChanged, quoteChanged, textChanged bool) error {
	switch mode {
	case config.ModeTime:
		if wordsChanged || quoteChanged || textChanged {
			return usageError("time mode only accepts --duration")
		}
	case config.ModeWords:
		if durationChanged || quoteChanged || textChanged {
			return usageError("words mode only accepts --words")
		}
	case config.ModeQuote:
		if durationChanged || wordsChanged || textChanged {
			return usageError("quote mode only accepts --quote-len")
		}
	case config.ModeCode:
		if durationChanged || wordsChanged || quoteChanged {
			return usageError("code mode only accepts --text")
		}
		if !textChanged || f.text == "" {
			return usageError("--text required for code mode")
		}
	}
	if f.json && !f.noTUI {
		return usageError("--json requires --no-tui")
	}
	return nil
}

func runLengthForMode(mode config.Mode, f runFlags, s config.Settings, durationChanged, wordsChanged bool) (int, error) {
	switch mode {
	case config.ModeTime:
		if durationChanged {
			if f.duration <= 0 {
				return 0, usageError("--duration must be positive")
			}
			return f.duration, nil
		}
	case config.ModeWords:
		if wordsChanged {
			if f.words <= 0 {
				return 0, usageError("--words must be positive")
			}
			return f.words, nil
		}
	}
	if mode == s.DefaultMode && containsInt(config.LengthsFor(mode), s.DefaultLength) {
		return s.DefaultLength, nil
	}
	switch mode {
	case config.ModeWords:
		return 25, nil
	case config.ModeTime:
		return 30, nil
	default:
		return 0, nil
	}
}

func validMode(mode config.Mode) bool {
	switch mode {
	case config.ModeTime, config.ModeWords, config.ModeQuote, config.ModeCode:
		return true
	default:
		return false
	}
}

func parseQuoteLen(s string) (words.QuoteLen, error) {
	switch s {
	case "", "medium":
		return words.QuoteMedium, nil
	case "short":
		return words.QuoteShort, nil
	case "long":
		return words.QuoteLong, nil
	default:
		return words.QuoteMedium, fmt.Errorf("invalid quote length %q", s)
	}
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func containsString(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
