package buff

import (
	"math"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

func TestCalculateMultiplier_Additive(t *testing.T) {
	got := CalculateMultiplier(50, RateMapping{RateType: "exp", Conversion: ConversionAdditive})
	if got != 1.5 {
		t.Errorf("CalculateMultiplier(50, Additive) = %v, want 1.5", got)
	}
}

func TestCalculateMultiplier_Direct(t *testing.T) {
	got := CalculateMultiplier(103, RateMapping{RateType: "meso", Conversion: ConversionDirect})
	if got != 1.03 {
		t.Errorf("CalculateMultiplier(103, Direct) = %v, want 1.03", got)
	}
}

func TestCalculateMultiplier_Fixed_IgnoresAmount(t *testing.T) {
	mapping := RateMapping{RateType: "exp", Conversion: ConversionFixed, Multiplier: 0.5}
	tests := []struct {
		name   string
		amount int32
	}{
		{"zero", 0},
		{"one", 1},
		{"fifty", 50},
		{"negative", -1},
		{"max", math.MaxInt32},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateMultiplier(tt.amount, mapping)
			if got != 0.5 {
				t.Errorf("CalculateMultiplier(%d, Fixed{0.5}) = %v, want 0.5", tt.amount, got)
			}
		})
	}
}

func TestCalculateMultiplier_FixedZeroMultiplier(t *testing.T) {
	got := CalculateMultiplier(50, RateMapping{RateType: "exp", Conversion: ConversionFixed, Multiplier: 0.0})
	if got != 0.0 {
		t.Errorf("CalculateMultiplier(_, Fixed{0.0}) = %v, want 0.0", got)
	}
}

func TestCalculateMultiplier_UnknownConversion(t *testing.T) {
	got := CalculateMultiplier(50, RateMapping{RateType: "exp", Conversion: ConversionMethod(999)})
	if got != 1.0 {
		t.Errorf("CalculateMultiplier(_, unknown) = %v, want 1.0", got)
	}
}

func TestBuffToRateMappings_Curse(t *testing.T) {
	mapping, exists := GetRateMapping(character.TemporaryStatTypeCurse)
	if !exists {
		t.Fatalf("GetRateMapping(%q) returned exists=false, want true", character.TemporaryStatTypeCurse)
	}
	if mapping.RateType != "exp" {
		t.Errorf("RateType = %q, want %q", mapping.RateType, "exp")
	}
	if mapping.Conversion != ConversionFixed {
		t.Errorf("Conversion = %v, want ConversionFixed", mapping.Conversion)
	}
	if mapping.Multiplier != 0.5 {
		t.Errorf("Multiplier = %v, want 0.5", mapping.Multiplier)
	}
}

func TestIsRateStatType_Curse(t *testing.T) {
	if !IsRateStatType(character.TemporaryStatTypeCurse) {
		t.Errorf("IsRateStatType(%q) = false, want true", character.TemporaryStatTypeCurse)
	}
}

func TestBuffToRateMappings_HolySymbolUnchanged(t *testing.T) {
	mapping, exists := GetRateMapping(character.TemporaryStatTypeHolySymbol)
	if !exists {
		t.Fatalf("GetRateMapping(%q) returned exists=false", character.TemporaryStatTypeHolySymbol)
	}
	if mapping.RateType != "exp" {
		t.Errorf("RateType = %q, want %q", mapping.RateType, "exp")
	}
	if mapping.Conversion != ConversionAdditive {
		t.Errorf("Conversion = %v, want ConversionAdditive", mapping.Conversion)
	}
}

func TestBuffToRateMappings_MesoUpUnchanged(t *testing.T) {
	mapping, exists := GetRateMapping(character.TemporaryStatTypeMesoUp)
	if !exists {
		t.Fatalf("GetRateMapping(%q) returned exists=false", character.TemporaryStatTypeMesoUp)
	}
	if mapping.RateType != "meso" {
		t.Errorf("RateType = %q, want %q", mapping.RateType, "meso")
	}
	if mapping.Conversion != ConversionDirect {
		t.Errorf("Conversion = %v, want ConversionDirect", mapping.Conversion)
	}
}

func TestIsRateStatType_PoisonNotMapped(t *testing.T) {
	// POISON is a stat-flag disease but is intentionally not in the rate mapping table
	// per PRD non-goals. Locking this in protects against accidental future expansion.
	if IsRateStatType(character.TemporaryStatTypePoison) {
		t.Errorf("IsRateStatType(%q) = true, want false", character.TemporaryStatTypePoison)
	}
}

// FR-A1/FR-A10: a configured multiplier of 2.0 is carried as amount = 200 and
// must read back as exactly 2.0x. ConversionDirect is amount/100.0, so the
// mapping is exactly invertible, which is what lets the multiplier live in
// configuration and still be displayed in the UI (FR-UI8).
func TestAnniversaryStatsConvertDirectly(t *testing.T) {
	for _, tc := range []struct {
		stat     character.TemporaryStatType
		rateType string
	}{
		{character.TemporaryStatTypeExpBuffRate, "exp"},
		{character.TemporaryStatTypeItemUpByItem, "item_drop"},
	} {
		m, ok := GetRateMapping(tc.stat)
		if !ok {
			t.Fatalf("%s has no rate mapping", tc.stat)
		}
		if m.RateType != tc.rateType {
			t.Fatalf("%s -> %q, want %q", tc.stat, m.RateType, tc.rateType)
		}
		if m.Conversion != ConversionDirect {
			t.Fatalf("%s conversion = %v, want ConversionDirect", tc.stat, m.Conversion)
		}
		if got := CalculateMultiplier(200, m); got != 2.0 {
			t.Fatalf("%s: amount 200 -> %vx, want 2.0x", tc.stat, got)
		}
	}
}

// EVENT_RATE is deliberately NOT mapped: it is a member of the JMS
// movement-affecting stat set, so buffing it would interact with the client's
// movement filter for no gameplay benefit over EXP_BUFF_RATE (design §10.3).
func TestEventRateIsNotMapped(t *testing.T) {
	if IsRateStatType(character.TemporaryStatTypeEventRate) {
		t.Fatalf("EVENT_RATE must not be mapped — see design §10.3")
	}
}
