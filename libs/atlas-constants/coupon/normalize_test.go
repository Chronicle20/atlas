package coupon

import "testing"

func TestNormalize(t *testing.T) {
	for _, c := range []struct{ name, in, want string }{
		{"uppercases", "maple2026", "MAPLE2026"},
		{"trims both ends", "  MAPLE2026  ", "MAPLE2026"},
		{"trims tabs and newlines", "\t MAPLE2026 \r\n", "MAPLE2026"},
		{"leaves an already-normal code alone", "MAPLE2026", "MAPLE2026"},
		{"does not strip interior spaces", "MAPLE 2026", "MAPLE 2026"},
		{"empty stays empty", "   ", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := Normalize(c.in); got != c.want {
				t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestPlausible(t *testing.T) {
	for _, c := range []struct {
		name string
		in   string
		want bool
	}{
		{"normal code", "MAPLE2026", true},
		{"single character", "A", true},
		{"exactly the column limit", "12345678901234567890123456789012", true},
		{"empty", "", false},
		{"one over the column limit", "123456789012345678901234567890123", false},
		{"un-normalized input is not plausible", " maple ", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := Plausible(c.in); got != c.want {
				t.Errorf("Plausible(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
