package ui

import "github.com/bavanchun/Typeburn/v2/internal/storage"

// avgWindow is how many recent runs the rail averages. Ten is short enough to
// move with the user's current form and long enough that one bad run does not
// dominate it.
const avgWindow = 10

// ResultContext is everything the comparison rail needs, resolved before the
// Result screen is built. The screen never sees []storage.Record: a sub-model
// holding the whole history would be one more place that could decide what a
// personal best is, and there is exactly one such place already.
//
// Every figure is scoped to the run's own mode+length bucket, so a 60-second
// best never answers a question about a 15-second run.
type ResultContext struct {
	HasHistory bool    // the bucket held at least one earlier run
	PB         float64 // best effective WPM in the bucket, excluding this run
	Avg10      float64 // mean effective WPM over the bucket's most recent runs
	Rank       int     // 1-based standing of this run inside the bucket
	Total      int     // bucket size counting this run
}

// ResultContextFor derives the comparison figures for a finished run from the
// history that was on disk before it was written.
//
// Eligibility matches the new-best rule exactly (see storage.EligibleForBest),
// so the rail and the ★ badge can never disagree about what a personal best is.
// A run that cannot hold a best — Code mode, or a letter-strict run whose cursor
// could not pass an error — has no comparable bucket and reads as a first run.
//
// hist must not already contain the finished run.
func ResultContextFor(hist []storage.Record, rec storage.Record) ResultContext {
	if !storage.EligibleForBest(rec) {
		return ResultContext{Rank: 1, Total: 1}
	}
	key := storage.BestBucketKey(rec.Mode, rec.Length)
	bucket := make([]float64, 0, len(hist))
	for _, h := range hist {
		if storage.EligibleForBest(h) && storage.BestBucketKey(h.Mode, h.Length) == key {
			bucket = append(bucket, storage.EffectiveWPM(h))
		}
	}

	ctx := ResultContext{Rank: 1, Total: len(bucket) + 1}
	if len(bucket) == 0 {
		return ctx
	}
	ctx.HasHistory = true

	mine := storage.EffectiveWPM(rec)
	for _, wpm := range bucket {
		if wpm > ctx.PB {
			ctx.PB = wpm
		}
		// Ties rank ahead of this run, so a repeated score reads as "no better
		// than last time" rather than promoting itself past its own equal.
		if wpm >= mine {
			ctx.Rank++
		}
	}

	window := bucket
	if len(window) > avgWindow {
		window = window[len(window)-avgWindow:]
	}
	recent := 0.0
	for _, wpm := range window {
		recent += wpm
	}
	ctx.Avg10 = recent / float64(len(window))
	return ctx
}

// Unranked drops this run's standing while keeping the figures that describe
// the history itself. It is what a run withheld from history gets: the personal
// best and recent average are still true, but the run took no place among them.
func (c ResultContext) Unranked() ResultContext {
	c.Rank, c.Total = 0, 0
	return c
}

// WithContext attaches the comparison figures the rail renders. Without it the
// rail shows its first-run form, which is also what a brand-new profile sees.
func (m ResultModel) WithContext(ctx ResultContext) ResultModel {
	m.ctx = ctx
	return m
}
