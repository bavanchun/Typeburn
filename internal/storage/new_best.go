package storage

import "strconv"

// BestBucketKey returns the comparison key for new-best scoping.
// For time and words modes the key includes the length parameter so that, e.g.,
// a 60s best does not suppress a 30s best. Quote mode has no numeric length
// (length is always 0) so the key is just "quote".
func BestBucketKey(mode string, length int) string {
	switch mode {
	case "time", "words":
		// Include length so time/30 and time/60 are separate leaderboards.
		key := mode + "/"
		key += strconv.Itoa(length)
		return key
	default:
		return mode
	}
}

// EffectiveWPM returns the effective WPM for new-best comparison as a float64.
// Post-fix records carry NetWPM directly. Legacy records (written before this
// field existed) unmarshal NetWPM as 0.0, so we fall back to float64(WPM) to
// keep the same integer scale — otherwise a new 60.x run would falsely beat a
// stored legacy 80. The 0.0-vs-legacy ambiguity is benign: a genuine run that
// yields NetWPM 0.0 also has WPM == 0, so float64(WPM) returns the correct 0.
func EffectiveWPM(r Record) float64 {
	if r.NetWPM == 0 {
		return float64(r.WPM)
	}
	return r.NetWPM
}

// BestWPMPerBucket returns the highest effective WPM for each mode+length bucket.
func BestWPMPerBucket(records []Record) map[string]float64 {
	bests := make(map[string]float64)
	for _, r := range records {
		key := BestBucketKey(r.Mode, r.Length)
		eff := EffectiveWPM(r)
		if prev, ok := bests[key]; !ok || eff > prev {
			bests[key] = eff
		}
	}
	return bests
}

// IsNewBest reports whether r represents a new personal best for its mode+length
// bucket compared to all records in hist.
//
// Rules:
//   - First-ever result for the bucket (no prior records) → true.
//   - Effective WPM of r strictly greater than every prior record's effective WPM
//     in the same bucket → true.
//   - Effective WPM of r equal to or less than the existing maximum → false.
//
// Effective WPM uses the stored NetWPM float when present, falling back to
// float64(WPM) for legacy records so scale comparison stays consistent.
//
// hist must NOT already contain r; call IsNewBest before AppendHistory.
// This function is pure and does not mutate hist.
func IsNewBest(hist []Record, r Record) bool {
	// Code-mode runs and strict runs are never personal bests (display-only, no leaderboard).
	if r.Mode == "code" || r.Strict {
		return false
	}
	key := BestBucketKey(r.Mode, r.Length)
	best := -1.0
	for _, h := range hist {
		if BestBucketKey(h.Mode, h.Length) == key {
			if eff := EffectiveWPM(h); eff > best {
				best = eff
			}
		}
	}
	// No prior record for this bucket → first result is always a new best.
	if best < 0 {
		return true
	}
	return EffectiveWPM(r) > best
}
