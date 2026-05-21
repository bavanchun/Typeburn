package update

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"v2.0.0", "v2.1.0", -1},
		{"v2.1.0", "v2.0.0", 1},
		{"v2.10.0", "v2.9.0", 1},
		{"v2.0.0", "v2.0.1", -1},
		{"v2.0.1", "v2.0.0", 1},
		{"v2.0.0-1-gabc123", "v2.0.0", 0}, // git-describe treated as v2.0.0
		{"v2.0.0", "v2.0.0-1-gabc123", 0}, // symmetric
		{"", "v2.0.0", 0},                 // malformed → equal
		{"v2.0.0", "", 0},                 // malformed → equal
		{"notaversion", "v2.0.0", 0},      // malformed → equal
		{"v2.0.0-rc.1", "v2.0.0", 0},      // suffix stripped; prerelease filtered by IsPrerelease
		{"v1.2.3", "v1.2.4", -1},
		{"v1.2.4", "v1.2.3", 1},
	}
	for _, tc := range cases {
		got := Compare(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
