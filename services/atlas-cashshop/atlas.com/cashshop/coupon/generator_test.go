package coupon

import (
	"errors"
	"strings"
	"testing"

	couponrules "github.com/Chronicle20/atlas/libs/atlas-constants/coupon"
)

func TestGenerateCodeUsesTheUnambiguousAlphabet(t *testing.T) {
	// No O/0, no I/1/L — these codes get read off a screen and typed by hand.
	banned := "O0I1L"
	for i := 0; i < 500; i++ {
		c, err := GenerateCode(10)
		if err != nil {
			t.Fatal(err)
		}
		if len(c) != 10 {
			t.Fatalf("len = %d, want 10", len(c))
		}
		if strings.ContainsAny(c, banned) {
			t.Fatalf("generated %q contains an ambiguous character", c)
		}
		if c != couponrules.Normalize(c) {
			t.Fatalf("generated %q is not already normalized", c)
		}
	}
}

func TestGenerateCodeIsNotObviouslyPredictable(t *testing.T) {
	// Codes are secrets; math/rand would make them guessable. This is a smoke
	// test for "not a fixed sequence", not a statistical test.
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		c, err := GenerateCode(12)
		if err != nil {
			t.Fatal(err)
		}
		if seen[c] {
			t.Fatalf("duplicate %q within 200 draws at length 12", c)
		}
		seen[c] = true
	}
}

func TestGenerateCodeRejectsOutOfRangeLengths(t *testing.T) {
	for _, length := range []int{0, -1, couponrules.MaxCodeLength + 1} {
		if _, err := GenerateCode(length); !errors.Is(err, ErrInvalidCoupon) {
			t.Errorf("GenerateCode(%d) error = %v, want ErrInvalidCoupon", length, err)
		}
	}
	if _, err := GenerateCode(couponrules.MaxCodeLength); err != nil {
		t.Errorf("GenerateCode(%d) = %v, want the column width to be accepted", couponrules.MaxCodeLength, err)
	}
}
