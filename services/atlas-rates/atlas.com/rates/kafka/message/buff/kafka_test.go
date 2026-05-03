package buff

import "testing"

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
