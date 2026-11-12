package store

import "testing"

func TestEscapeLike(t *testing.T) {
	cases := map[string]string{
		"rtx":     "rtx",
		"100%":    `100\%`,
		"a_b":     `a\_b`,
		`a\b`:     `a\\b`,
		`%_\`:     `\%\_\\`,
		"การ์ดจอ": "การ์ดจอ",
	}
	for in, want := range cases {
		if got := escapeLike(in); got != want {
			t.Errorf("escapeLike(%q) = %q, want %q", in, got, want)
		}
	}
}
