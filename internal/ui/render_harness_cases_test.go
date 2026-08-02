package ui

import (
	"time"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/storage"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/words"
)

// settledTyping renders the typing screen from a fixed seed, so the generated
// words — and therefore the line count — are the same on every run.
//
// The clock needs care and does not have a seam. TypingModel.View calls
// elapsedMs(m.startMs, time.Now()) directly; the nowFn field is read only by
// the input path and never reaches View. These frames are stable because an
// untouched model has startMs == 0, which elapsedMs short-circuits to 0, and
// because the caret reads the nowMs field rather than the clock. A case that
// applies keystrokes would set startMs and become wall-clock dependent — freeze
// startMs and nowMs explicitly if one is ever added.
func settledTyping(mode config.Mode, length int, ql words.QuoteLen) func(theme.Theme, int, int) string {
	return func(th theme.Theme, w, h int) string {
		m := newTypingWithSeed(mode, length, ql, th, config.DefaultKeymap(), false, false, false, false, 42)
		return m.SetSize(w, h).View()
	}
}

// harnessResult is a run with a value that exercises the block-digit glyphs
// (a 0 and a ragged digit) rather than a value that happens to render cleanly.
func harnessResult() ResultMsg {
	per := make([]metrics.PerSecond, 30)
	for i := range per {
		per[i] = metrics.PerSecond{
			Sec: i, RawWPM: float64(70 + i%13), CorrectChars: 5, TotalChars: 5,
		}
		if i%5 == 0 {
			per[i].Errors = 1
		}
	}
	return ResultMsg{
		Result: metrics.Result{
			NetWPM: 106, RawWPM: 112, Accuracy: 96.4, Consistency: 83.2,
			CorrectChars: 268, DurationMs: 30000, PerSecond: per,
		},
		Mode:   config.ModeTime,
		Length: 30,
	}
}

// harnessRecords builds n history records with values wide enough to exercise
// the table's column widths rather than its empty state.
func harnessRecords(n int) []storage.Record {
	recs := make([]storage.Record, n)
	base := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)
	modes := []string{"time", "words", "quote", "code"}
	for i := range recs {
		recs[i] = storage.Record{
			Time:        base.Add(time.Duration(i) * time.Hour),
			Mode:        modes[i%len(modes)],
			Length:      []int{15, 30, 60, 120}[i%4],
			WPM:         60 + i%80,
			NetWPM:      float64(60 + i%80),
			RawWPM:      float64(66 + i%80),
			Accuracy:    90 + float64(i%10),
			Consistency: 70 + float64(i%25),
		}
	}
	return recs
}

// cjkSample mixes wide CJK, an emoji and ASCII. Without a case like this the
// frame-fits assertions cannot see a display-width defect: code that counts
// runes where it means cells still measures "narrow enough" on pure ASCII.
const cjkSample = "函数 返回 一个 值 🚀 done 我们 需要 更 多 的 宽 字 符 来 填 满 这 一 行 test"

// screenCases covers all six screens, the states that change their geometry,
// and a wide-rune fixture. Every frame-fits assertion runs over this list, so
// adding a case here extends every invariant at once.
func screenCases() []screenCase {
	km := config.DefaultKeymap()
	set := config.Defaults()

	return []screenCase{
		{"home", func(th theme.Theme, w, h int) string {
			return NewHome(set, th, km, "", "").SetSize(w, h).View()
		}},
		{"home/code-loaded", func(th theme.Theme, w, h int) string {
			return NewHome(set, th, km, "package main", "").SetSize(w, h).View()
		}},
		{"home/code-error", func(th theme.Theme, w, h int) string {
			return NewHome(set, th, km, "", "file is empty after normalization").SetSize(w, h).View()
		}},
		{"typing/time30", settledTyping(config.ModeTime, 30, words.QuoteShort)},
		{"typing/words25", settledTyping(config.ModeWords, 25, words.QuoteShort)},
		{"typing/quote", settledTyping(config.ModeQuote, 0, words.QuoteMedium)},
		{"typing/code", func(th theme.Theme, w, h int) string {
			return NewTypingCode("func main() {\n\tprintln(\"hi\")\n}\n", th, km, false, false).SetSize(w, h).View()
		}},
		{"typing/cjk", func(th theme.Theme, w, h int) string {
			return NewTypingCode(cjkSample, th, km, false, false).SetSize(w, h).View()
		}},
		{"result", func(th theme.Theme, w, h int) string {
			m := NewResult(harnessResult(), th, km).SetSize(w, h)
			m.revealStartMs, m.nowMs = 0, 1<<40
			return m.View()
		}},
		{"settings", func(th theme.Theme, w, h int) string {
			return NewSettings(set, th, km).SetSize(w, h).View()
		}},
		{"history/empty", func(th theme.Theme, w, h int) string {
			return NewHistory(nil, th, km).SetSize(w, h).View()
		}},
		{"history/120", func(th theme.Theme, w, h int) string {
			return NewHistory(harnessRecords(120), th, km).SetSize(w, h).View()
		}},
		{"codepaste", func(th theme.Theme, w, h int) string {
			return NewCodePaste(th).SetSize(w, h).View()
		}},
		{"codepaste/error", func(th theme.Theme, w, h int) string {
			m := NewCodePaste(th).SetSize(w, h)
			m.errMsg = "pasted text is empty after normalization"
			return m.View()
		}},
	}
}
