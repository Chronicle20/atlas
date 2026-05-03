package buff

import (
	"math"
	"testing"
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
	mapping, exists := GetRateMapping(StatTypeCurse)
	if !exists {
		t.Fatalf("GetRateMapping(%q) returned exists=false, want true", StatTypeCurse)
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
	if !IsRateStatType(StatTypeCurse) {
		t.Errorf("IsRateStatType(%q) = false, want true", StatTypeCurse)
	}
}
