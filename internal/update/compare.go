package update

import (
	"strconv"
	"strings"
)

// Compare returns -1 if a < b, 0 if a == b, +1 if a > b.
// Strips leading "v". Treats git-describe form (vX.Y.Z-N-gSHA) as vX.Y.Z.
// Returns 0 for malformed input (treat as equal; caller filters with IsPrerelease).
func Compare(a, b string) int {
	ma, ok1 := parseSemver(a)
	mb, ok2 := parseSemver(b)
	if !ok1 || !ok2 {
		return 0
	}
	for i := range ma {
		if ma[i] < mb[i] {
			return -1
		}
		if ma[i] > mb[i] {
			return 1
		}
	}
	return 0
}

// parseSemver parses "vX.Y.Z" or "vX.Y.Z-<suffix>" into [major, minor, patch].
// Any suffix (prerelease label or git-describe) is stripped — comparisons use
// only the numeric core so that prerelease filtering happens in IsPrerelease.
func parseSemver(v string) ([3]int, bool) {
	v = strings.TrimPrefix(v, "v")
	core := strings.SplitN(v, "-", 2)[0]
	segs := strings.Split(core, ".")
	if len(segs) != 3 {
		return [3]int{}, false
	}
	var nums [3]int
	for i, s := range segs {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return [3]int{}, false
		}
		nums[i] = n
	}
	return nums, true
}
