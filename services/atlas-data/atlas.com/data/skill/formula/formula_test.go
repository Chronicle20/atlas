package formula

import (
	"strings"
	"testing"
)

func TestEvaluate(t *testing.T) {
	testCases := []struct {
		name  string
		src   string
		level int
		want  int64
	}{
		// FR-3.4 — division is real, not integer.
		{"ceil half at level 1", "u(x/2)", 1, 1},
		{"floor quarter at level 1", "d(x/4)", 1, 0},
		{"floor quarter at level 4", "d(x/4)", 4, 1},
		// design §1.3 — u/d are trunc-based, not ceil/floor. The 0.999999
		// fudge must not push an exact integer up.
		{"exact integer does not round up", "u(x/5)", 5, 1},
		{"negative floor truncates toward zero", "d(-x/2)", 1, 0},
		{"negative ceiling truncates toward zero", "u(-x/2)", 3, 0},
		// design §1.4 — precedence is + -> - -> / -> *, so `*` binds
		// tighter than `/`. Left-to-right would give 30, not 3.
		{"star binds tighter than slash", "x/2*3", 20, 3},
		// FR-3.5 — unary minus.
		{"bare negative literal", "-2", 1, -2},
		{"leading negative expression", "-10-1*x", 3, -13},
		// FR-3.6 — the single decimal literal in the archive.
		{"decimal truncates at level 1", "0.5*x", 1, 0},
		{"decimal truncates at level 3", "0.5*x", 3, 1},
		{"decimal with addition", "5+0.5*x", 5, 7},
		// FR-3.7 — the one whitespace-bearing value (skill 2111002 damage).
		{"leading space", " 375+5*x", 1, 380},
		// FR-3.9 — maximum observed complexity and longest value.
		{"four operators at level 1", "-1-1*u(x/10)", 1, -2},
		{"four operators at level 20", "-1-1*u(x/10)", 20, -3},
		{"longest value at level 1", "150+50*u(x/10)", 1, 200},
		{"longest value at level 20", "150+50*u(x/10)", 20, 250},
		// FR-3.8 / design §4.4 — deliberate superset of the client.
		{"nested calls", "u(d(x/2))", 5, 2},
		{"bare parentheses truncate", "(x/2)*3", 5, 6},
		// design §1.1 — evaluation is case-insensitive (_strlwr).
		{"uppercase source", "U(X/2)", 1, 1},
		// Constant sources: an int-typed leaf reaches Parse as decimal text.
		{"plain integer", "0", 7, 0},
		{"plain integer one", "1", 7, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e, err := Parse(tc.src)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want nil", tc.src, err)
			}
			got, err := e.Evaluate(tc.level)
			if err != nil {
				t.Fatalf("Evaluate(%d) error = %v, want nil", tc.level, err)
			}
			if got != tc.want {
				t.Fatalf("Parse(%q).Evaluate(%d) = %d, want %d", tc.src, tc.level, got, tc.want)
			}
		})
	}
}

func TestParseRejects(t *testing.T) {
	testCases := []struct {
		name string
		src  string
	}{
		{"empty", ""},
		{"blank", "   "},
		{"unknown identifier", "y+1"},
		{"unknown multi-letter identifier", "level+1"},
		{"unknown function", "f(x)"},
		{"function arity two", "u(x,1)"},
		{"unbalanced open paren", "u(x/2"},
		{"unbalanced close paren", "x/2)"},
		{"trailing operator", "x+"},
		{"double operator", "x**2"},
		{"bare function name", "u"},
		{"over length", strings.Repeat("1+", 200) + "1"},
		{"too many tokens", strings.Repeat("1+", 40) + "1"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(tc.src); err == nil {
				t.Fatalf("Parse(%q) error = nil, want a parse error", tc.src)
			}
		})
	}
}

func TestEvaluateRejectsNonFinite(t *testing.T) {
	e, err := Parse("x/0")
	if err != nil {
		t.Fatalf("Parse error = %v, want nil", err)
	}
	if _, err := e.Evaluate(1); err == nil {
		t.Fatal("Evaluate error = nil, want a non-finite result error")
	}
}

func TestParseOnceEvaluateMany(t *testing.T) {
	e, err := Parse("6+2*u(x/5)")
	if err != nil {
		t.Fatalf("Parse error = %v, want nil", err)
	}
	want := map[int]int64{1: 8, 5: 8, 6: 10, 20: 14}
	for level, w := range want {
		got, err := e.Evaluate(level)
		if err != nil {
			t.Fatalf("Evaluate(%d) error = %v", level, err)
		}
		if got != w {
			t.Fatalf("Evaluate(%d) = %d, want %d", level, got, w)
		}
	}
	if e.Source() != "6+2*u(x/5)" {
		t.Fatalf("Source() = %q, want %q", e.Source(), "6+2*u(x/5)")
	}
}
