package heal

import (
	"math"
	"testing"
)

func TestEffectiveMaxHpOrBase_PrefersEffective(t *testing.T) {
	got := effectiveMaxHpOrBase(12000, 5000)
	if got != 12000 {
		t.Fatalf("effectiveMaxHpOrBase(12000, 5000) = %d, want 12000", got)
	}
}

func TestEffectiveMaxHpOrBase_ZeroEffectiveFallsBackToBase(t *testing.T) {
	got := effectiveMaxHpOrBase(0, 5000)
	if got != 5000 {
		t.Fatalf("effectiveMaxHpOrBase(0, 5000) = %d, want 5000 (fallback)", got)
	}
}

func TestEffectiveMaxHpOrBase_OverUint16ClampsToMax(t *testing.T) {
	got := effectiveMaxHpOrBase(70000, 5000)
	if got != math.MaxUint16 {
		t.Fatalf("effectiveMaxHpOrBase(70000, 5000) = %d, want %d (uint16 max)", got, math.MaxUint16)
	}
}
