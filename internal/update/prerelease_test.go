package update

import "testing"

func TestIsPrerelease(t *testing.T) {
	cases := []struct {
		tag  string
		want bool
	}{
		{"v2.0.0", false},
		{"v2.1.0", false},
		{"v2.0.0-rc.1", true},
		{"v2.0.0-rc1", true},
		{"v0.0.0-rc.test", true},
		{"V2.0.0-Alpha", true}, // case-insensitive
		{"v2.0.0-beta.3", true},
		{"v2.0.0-alpha", true},
		{"v2.0.0-pre.1", true},
		{"v2.0.0-7-gabc123", false}, // git-describe, NOT a prerelease
		{"v2.0.0-3-gdeadbeef", false},
		{"v2.0.0-1", true},        // numeric suffix only — SemVer-2 prerelease
		{"v2.0.0-canary.4", true}, // canary build
		{"v0.0.0-test", true},     // v0.0.0- prefix
	}
	for _, tc := range cases {
		got := IsPrerelease(tc.tag)
		if got != tc.want {
			t.Errorf("IsPrerelease(%q) = %v, want %v", tc.tag, got, tc.want)
		}
	}
}
