package pet_test

import (
	"errors"
	"strings"
	"testing"

	petconst "github.com/Chronicle20/atlas/libs/atlas-constants/pet"
)

func TestNormalizeNameTrimsSurroundingWhitespace(t *testing.T) {
	if got := petconst.NormalizeName("  Fluffy \t"); got != "Fluffy" {
		t.Fatalf("NormalizeName = %q, want %q", got, "Fluffy")
	}
}

func TestValidateNameAcceptsBounds(t *testing.T) {
	for _, name := range []string{"Abcd", "Abcdefghijkl"} {
		if err := petconst.ValidateName(name); err != nil {
			t.Fatalf("ValidateName(%q) = %v, want nil", name, err)
		}
	}
}

func TestValidateNameRejectsTooShort(t *testing.T) {
	for _, name := range []string{"", "Abc"} {
		if err := petconst.ValidateName(name); !errors.Is(err, petconst.ErrNameTooShort) {
			t.Fatalf("ValidateName(%q) = %v, want ErrNameTooShort", name, err)
		}
	}
}

func TestValidateNameRejectsTooLong(t *testing.T) {
	name := strings.Repeat("A", petconst.MaxNameLength+1)
	if err := petconst.ValidateName(name); !errors.Is(err, petconst.ErrNameTooLong) {
		t.Fatalf("ValidateName(%q) = %v, want ErrNameTooLong", name, err)
	}
}

// A whitespace-only name is rejected only because the caller normalized first;
// ValidateName itself never trims (PRD FR-4.2/FR-4.3).
func TestNormalizeThenValidateRejectsWhitespaceOnly(t *testing.T) {
	if err := petconst.ValidateName(petconst.NormalizeName("     ")); !errors.Is(err, petconst.ErrNameTooShort) {
		t.Fatalf("whitespace-only name accepted")
	}
}

func TestBoundsMatchClientDialog(t *testing.T) {
	if petconst.MinNameLength != 4 || petconst.MaxNameLength != 12 {
		t.Fatalf("bounds = (%d,%d), want (4,12) per sub_9AC7CB @0xa0bb2f", petconst.MinNameLength, petconst.MaxNameLength)
	}
}
