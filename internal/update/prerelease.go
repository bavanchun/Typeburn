package update

import "strings"

// IsPrerelease reports whether a release tag should be excluded from stable
// upgrade comparisons. The API's Prerelease/Draft fields are the primary signal;
// this function is a belt-and-suspenders tag-name heuristic for any gaps.
func IsPrerelease(tag string) bool {
	lower := strings.ToLower(tag)
	for _, label := range []string{"-rc", "-beta", "-alpha", "-pre"} {
		if strings.Contains(lower, label) {
			return true
		}
	}
	if strings.HasPrefix(lower, "v0.0.0-") {
		return true
	}
	// Any suffix after MAJOR.MINOR.PATCH is treated as prerelease unless it
	// matches the git-describe form "N-gSHA" (e.g. "3-gabc1234").
	v := strings.TrimPrefix(lower, "v")
	parts := strings.SplitN(v, "-", 2)
	if len(parts) == 2 && !isGitDescribeSuffix(parts[1]) {
		return true
	}
	return false
}

// isGitDescribeSuffix returns true if s matches the git-describe suffix form
// "N-gHEX" where N is a non-negative integer and gHEX is a "g" + hex string.
func isGitDescribeSuffix(s string) bool {
	dash := strings.Index(s, "-")
	if dash < 1 {
		return false
	}
	count, rest := s[:dash], s[dash+1:]
	for _, c := range count {
		if c < '0' || c > '9' {
			return false
		}
	}
	if len(rest) < 2 || rest[0] != 'g' {
		return false
	}
	for _, c := range rest[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
