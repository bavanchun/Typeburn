package storage

// modeKey returns the comparison key for new-best scoping.
// For time and words modes the key includes the length parameter so that, e.g.,
// a 60s best does not suppress a 30s best. Quote mode has no numeric length
// (length is always 0) so the key is just "quote".
func modeKey(mode string, length int) string {
	switch mode {
	case "time", "words":
		// Include length so time/30 and time/60 are separate leaderboards.
		key := mode + "/"
		// Inline int-to-string to avoid importing fmt/strconv in this tiny file.
		key += itoa(length)
		return key
	default:
		return mode
	}
}

// itoa converts a non-negative integer to its decimal string representation
// without importing fmt or strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

// IsNewBest reports whether r represents a new personal best for its mode+length
// bucket compared to all records in hist.
//
// Rules:
//   - First-ever result for the bucket (no prior records) → true.
//   - r.WPM strictly greater than every prior record's WPM in the same bucket → true.
//   - r.WPM equal to or less than the existing maximum → false.
//
// hist must NOT already contain r; call IsNewBest before AppendHistory.
// This function is pure and does not mutate hist.
func IsNewBest(hist []Record, r Record) bool {
	key := modeKey(r.Mode, r.Length)
	best := -1
	for _, h := range hist {
		if modeKey(h.Mode, h.Length) == key {
			if h.WPM > best {
				best = h.WPM
			}
		}
	}
	// No prior record for this bucket → first result is always a new best.
	if best < 0 {
		return true
	}
	return r.WPM > best
}
