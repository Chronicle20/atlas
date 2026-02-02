package rate

import (
	"testing"
)

func TestNewFactor(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		rateType   Type
		multiplier float64
	}{
		{"world factor", "world", TypeExp, 2.0},
		{"buff factor", "buff:2311003", TypeExp, 1.5},
		{"item factor", "item:1002357", TypeMeso, 1.2},
		{"zero multiplier", "test", TypeItemDrop, 0.0},
		{"negative multiplier", "test", TypeQuestExp, -1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFactor(tt.source, tt.rateType, tt.multiplier)

			if f.Source() != tt.source {
				t.Errorf("Source() = %v, want %v", f.Source(), tt.source)
			}
			if f.RateType() != tt.rateType {
				t.Errorf("RateType() = %v, want %v", f.RateType(), tt.rateType)
			}
			if f.Multiplier() != tt.multiplier {
				t.Errorf("Multiplier() = %v, want %v", f.Multiplier(), tt.multiplier)
			}
		})
	}
}

func TestAllTypes(t *testing.T) {
	types := AllTypes()

	if len(types) != 4 {
		t.Errorf("AllTypes() returned %d types, want 4", len(types))
	}

	expected := map[Type]bool{
		TypeExp:      false,
		TypeMeso:     false,
		TypeItemDrop: false,
		TypeQuestExp: false,
	}

	for _, rt := range types {
		if _, ok := expected[rt]; !ok {
			t.Errorf("AllTypes() contains unexpected type: %v", rt)
		}
		expected[rt] = true
	}

	for rt, found := range expected {
		if !found {
			t.Errorf("AllTypes() missing type: %v", rt)
		}
	}
}

func TestDefaultComputed(t *testing.T) {
	c := DefaultComputed()

	if c.ExpRate() != 1.0 {
		t.Errorf("ExpRate() = %v, want 1.0", c.ExpRate())
	}
	if c.MesoRate() != 1.0 {
		t.Errorf("MesoRate() = %v, want 1.0", c.MesoRate())
	}
	if c.ItemDropRate() != 1.0 {
		t.Errorf("ItemDropRate() = %v, want 1.0", c.ItemDropRate())
	}
	if c.QuestExpRate() != 1.0 {
		t.Errorf("QuestExpRate() = %v, want 1.0", c.QuestExpRate())
	}
}

func TestNewComputed(t *testing.T) {
	c := NewComputed(2.0, 3.0, 4.0, 5.0)

	if c.ExpRate() != 2.0 {
		t.Errorf("ExpRate() = %v, want 2.0", c.ExpRate())
	}
	if c.MesoRate() != 3.0 {
		t.Errorf("MesoRate() = %v, want 3.0", c.MesoRate())
	}
	if c.ItemDropRate() != 4.0 {
		t.Errorf("ItemDropRate() = %v, want 4.0", c.ItemDropRate())
	}
	if c.QuestExpRate() != 5.0 {
		t.Errorf("QuestExpRate() = %v, want 5.0", c.QuestExpRate())
	}
}

func TestComputedGetRate(t *testing.T) {
	c := NewComputed(2.0, 3.0, 4.0, 5.0)

	tests := []struct {
		rateType Type
		expected float64
	}{
		{TypeExp, 2.0},
		{TypeMeso, 3.0},
		{TypeItemDrop, 4.0},
		{TypeQuestExp, 5.0},
		{Type("unknown"), 1.0},
	}

	for _, tt := range tests {
		t.Run(string(tt.rateType), func(t *testing.T) {
			if got := c.GetRate(tt.rateType); got != tt.expected {
				t.Errorf("GetRate(%v) = %v, want %v", tt.rateType, got, tt.expected)
			}
		})
	}
}

func TestComputeFromFactors_Empty(t *testing.T) {
	c := ComputeFromFactors(nil)

	if c.ExpRate() != 1.0 {
		t.Errorf("ExpRate() = %v, want 1.0", c.ExpRate())
	}
	if c.MesoRate() != 1.0 {
		t.Errorf("MesoRate() = %v, want 1.0", c.MesoRate())
	}
	if c.ItemDropRate() != 1.0 {
		t.Errorf("ItemDropRate() = %v, want 1.0", c.ItemDropRate())
	}
	if c.QuestExpRate() != 1.0 {
		t.Errorf("QuestExpRate() = %v, want 1.0", c.QuestExpRate())
	}
}

func TestComputeFromFactors_SingleFactor(t *testing.T) {
	factors := []Factor{
		NewFactor("world", TypeExp, 2.0),
	}

	c := ComputeFromFactors(factors)

	if c.ExpRate() != 2.0 {
		t.Errorf("ExpRate() = %v, want 2.0", c.ExpRate())
	}
	if c.MesoRate() != 1.0 {
		t.Errorf("MesoRate() = %v, want 1.0", c.MesoRate())
	}
}

func TestComputeFromFactors_MultipleFactorsSameType(t *testing.T) {
	factors := []Factor{
		NewFactor("world", TypeExp, 2.0),
		NewFactor("buff:2311003", TypeExp, 1.5),
	}

	c := ComputeFromFactors(factors)

	// 2.0 * 1.5 = 3.0
	if c.ExpRate() != 3.0 {
		t.Errorf("ExpRate() = %v, want 3.0", c.ExpRate())
	}
}

func floatEquals(a, b, epsilon float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

func TestComputeFromFactors_MultipleFactorsDifferentTypes(t *testing.T) {
	factors := []Factor{
		NewFactor("world", TypeExp, 2.0),
		NewFactor("world", TypeMeso, 1.5),
		NewFactor("buff:4111001", TypeMeso, 1.2),
		NewFactor("item:5210000", TypeItemDrop, 2.0),
	}

	c := ComputeFromFactors(factors)

	if c.ExpRate() != 2.0 {
		t.Errorf("ExpRate() = %v, want 2.0", c.ExpRate())
	}
	// 1.5 * 1.2 = 1.8 (use approximate comparison for floating point)
	if !floatEquals(c.MesoRate(), 1.8, 0.0001) {
		t.Errorf("MesoRate() = %v, want approximately 1.8", c.MesoRate())
	}
	if c.ItemDropRate() != 2.0 {
		t.Errorf("ItemDropRate() = %v, want 2.0", c.ItemDropRate())
	}
	if c.QuestExpRate() != 1.0 {
		t.Errorf("QuestExpRate() = %v, want 1.0", c.QuestExpRate())
	}
}

func TestComputeFromFactors_OrderIndependence(t *testing.T) {
	factors1 := []Factor{
		NewFactor("world", TypeExp, 2.0),
		NewFactor("buff:2311003", TypeExp, 1.5),
		NewFactor("item:1002357", TypeExp, 1.1),
	}

	factors2 := []Factor{
		NewFactor("item:1002357", TypeExp, 1.1),
		NewFactor("world", TypeExp, 2.0),
		NewFactor("buff:2311003", TypeExp, 1.5),
	}

	c1 := ComputeFromFactors(factors1)
	c2 := ComputeFromFactors(factors2)

	if c1.ExpRate() != c2.ExpRate() {
		t.Errorf("Factor order affects result: %v != %v", c1.ExpRate(), c2.ExpRate())
	}
}

func TestComputeFromFactors_ZeroMultiplier(t *testing.T) {
	factors := []Factor{
		NewFactor("world", TypeExp, 2.0),
		NewFactor("debuff", TypeExp, 0.0),
	}

	c := ComputeFromFactors(factors)

	// 2.0 * 0.0 = 0.0
	if c.ExpRate() != 0.0 {
		t.Errorf("ExpRate() = %v, want 0.0", c.ExpRate())
	}
}

func TestComputeFromFactors_LargeMultipliers(t *testing.T) {
	factors := []Factor{
		NewFactor("event", TypeExp, 10.0),
		NewFactor("buff", TypeExp, 5.0),
	}

	c := ComputeFromFactors(factors)

	// 10.0 * 5.0 = 50.0
	if c.ExpRate() != 50.0 {
		t.Errorf("ExpRate() = %v, want 50.0", c.ExpRate())
	}
}

func TestComputeFromFactors_FractionalMultipliers(t *testing.T) {
	factors := []Factor{
		NewFactor("world", TypeMeso, 1.03),
		NewFactor("buff:4111001", TypeMeso, 1.2),
	}

	c := ComputeFromFactors(factors)

	// 1.03 * 1.2 = 1.236
	if !floatEquals(c.MesoRate(), 1.236, 0.0001) {
		t.Errorf("MesoRate() = %v, want approximately 1.236", c.MesoRate())
	}
}
