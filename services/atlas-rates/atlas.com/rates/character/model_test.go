package character

import (
	"atlas-rates/rate"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func createTestTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func TestNewModel(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)

	if m.Tenant() != ten {
		t.Errorf("Tenant() = %v, want %v", m.Tenant(), ten)
	}
	if m.WorldId() != 1 {
		t.Errorf("WorldId() = %v, want 1", m.WorldId())
	}
	if m.ChannelId() != 2 {
		t.Errorf("ChannelId() = %v, want 2", m.ChannelId())
	}
	if m.CharacterId() != 12345 {
		t.Errorf("CharacterId() = %v, want 12345", m.CharacterId())
	}
	if len(m.Factors()) != 0 {
		t.Errorf("Factors() length = %v, want 0", len(m.Factors()))
	}
}

func TestModelComputedRates_Empty(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)
	computed := m.ComputedRates()

	if computed.ExpRate() != 1.0 {
		t.Errorf("ExpRate() = %v, want 1.0", computed.ExpRate())
	}
	if computed.MesoRate() != 1.0 {
		t.Errorf("MesoRate() = %v, want 1.0", computed.MesoRate())
	}
	if computed.ItemDropRate() != 1.0 {
		t.Errorf("ItemDropRate() = %v, want 1.0", computed.ItemDropRate())
	}
	if computed.QuestExpRate() != 1.0 {
		t.Errorf("QuestExpRate() = %v, want 1.0", computed.QuestExpRate())
	}
}

func TestModelWithFactor_AddNew(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)

	f := rate.NewFactor("world", rate.TypeExp, 2.0)
	m2 := m.WithFactor(f)

	// Original should be unchanged (immutable)
	if len(m.Factors()) != 0 {
		t.Errorf("Original model modified: factors = %v, want 0", len(m.Factors()))
	}

	// New model should have the factor
	if len(m2.Factors()) != 1 {
		t.Errorf("New model factors = %v, want 1", len(m2.Factors()))
	}

	if m2.ComputedRates().ExpRate() != 2.0 {
		t.Errorf("ExpRate() = %v, want 2.0", m2.ComputedRates().ExpRate())
	}
}

func TestModelWithFactor_ReplaceSameSourceAndType(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)

	f1 := rate.NewFactor("world", rate.TypeExp, 2.0)
	m = m.WithFactor(f1)

	f2 := rate.NewFactor("world", rate.TypeExp, 3.0)
	m = m.WithFactor(f2)

	// Should still have only 1 factor
	if len(m.Factors()) != 1 {
		t.Errorf("Factors count = %v, want 1", len(m.Factors()))
	}

	// Should have the updated value
	if m.ComputedRates().ExpRate() != 3.0 {
		t.Errorf("ExpRate() = %v, want 3.0", m.ComputedRates().ExpRate())
	}
}

func TestModelWithFactor_DifferentSources(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)

	f1 := rate.NewFactor("world", rate.TypeExp, 2.0)
	f2 := rate.NewFactor("buff:2311003", rate.TypeExp, 1.5)
	m = m.WithFactor(f1).WithFactor(f2)

	if len(m.Factors()) != 2 {
		t.Errorf("Factors count = %v, want 2", len(m.Factors()))
	}

	// 2.0 * 1.5 = 3.0
	if m.ComputedRates().ExpRate() != 3.0 {
		t.Errorf("ExpRate() = %v, want 3.0", m.ComputedRates().ExpRate())
	}
}

func TestModelWithFactor_SameSourceDifferentTypes(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)

	f1 := rate.NewFactor("world", rate.TypeExp, 2.0)
	f2 := rate.NewFactor("world", rate.TypeMeso, 1.5)
	m = m.WithFactor(f1).WithFactor(f2)

	if len(m.Factors()) != 2 {
		t.Errorf("Factors count = %v, want 2", len(m.Factors()))
	}

	if m.ComputedRates().ExpRate() != 2.0 {
		t.Errorf("ExpRate() = %v, want 2.0", m.ComputedRates().ExpRate())
	}
	if m.ComputedRates().MesoRate() != 1.5 {
		t.Errorf("MesoRate() = %v, want 1.5", m.ComputedRates().MesoRate())
	}
}

func TestModelWithoutFactor_RemoveExisting(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)

	f := rate.NewFactor("world", rate.TypeExp, 2.0)
	m = m.WithFactor(f)

	m2 := m.WithoutFactor("world", rate.TypeExp)

	// Original should still have the factor (immutable)
	if len(m.Factors()) != 1 {
		t.Errorf("Original model modified: factors = %v, want 1", len(m.Factors()))
	}

	// New model should not have the factor
	if len(m2.Factors()) != 0 {
		t.Errorf("New model factors = %v, want 0", len(m2.Factors()))
	}

	if m2.ComputedRates().ExpRate() != 1.0 {
		t.Errorf("ExpRate() = %v, want 1.0", m2.ComputedRates().ExpRate())
	}
}

func TestModelWithoutFactor_RemoveNonExistent(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)

	f := rate.NewFactor("world", rate.TypeExp, 2.0)
	m = m.WithFactor(f)

	// Try to remove a factor that doesn't exist
	m2 := m.WithoutFactor("buff:123", rate.TypeExp)

	// Should still have the original factor
	if len(m2.Factors()) != 1 {
		t.Errorf("Factors count = %v, want 1", len(m2.Factors()))
	}
}

func TestModelWithoutFactor_PartialMatch(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)

	f := rate.NewFactor("world", rate.TypeExp, 2.0)
	m = m.WithFactor(f)

	// Try to remove with matching source but different type
	m2 := m.WithoutFactor("world", rate.TypeMeso)

	// Should still have the original factor
	if len(m2.Factors()) != 1 {
		t.Errorf("Factors count = %v, want 1", len(m2.Factors()))
	}

	// Try to remove with matching type but different source
	m3 := m.WithoutFactor("other", rate.TypeExp)

	if len(m3.Factors()) != 1 {
		t.Errorf("Factors count = %v, want 1", len(m3.Factors()))
	}
}

func TestModelWithoutFactorsBySource_RemoveAll(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)

	f1 := rate.NewFactor("world", rate.TypeExp, 2.0)
	f2 := rate.NewFactor("world", rate.TypeMeso, 1.5)
	f3 := rate.NewFactor("buff:123", rate.TypeExp, 1.2)
	m = m.WithFactor(f1).WithFactor(f2).WithFactor(f3)

	m2 := m.WithoutFactorsBySource("world")

	// Should only have the buff factor left
	if len(m2.Factors()) != 1 {
		t.Errorf("Factors count = %v, want 1", len(m2.Factors()))
	}

	factors := m2.Factors()
	if factors[0].Source() != "buff:123" {
		t.Errorf("Remaining factor source = %v, want buff:123", factors[0].Source())
	}
}

func TestModelWithoutFactorsBySource_NoMatch(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)

	f := rate.NewFactor("world", rate.TypeExp, 2.0)
	m = m.WithFactor(f)

	m2 := m.WithoutFactorsBySource("nonexistent")

	if len(m2.Factors()) != 1 {
		t.Errorf("Factors count = %v, want 1", len(m2.Factors()))
	}
}

func TestModelFactors_DefensiveCopy(t *testing.T) {
	ten := createTestTenant()
	m := NewModel(ten, 1, 2, 12345)

	f := rate.NewFactor("world", rate.TypeExp, 2.0)
	m = m.WithFactor(f)

	factors1 := m.Factors()
	factors2 := m.Factors()

	// Modify the first slice
	if len(factors1) > 0 {
		factors1[0] = rate.NewFactor("modified", rate.TypeMeso, 99.0)
	}

	// Second slice should be unaffected
	if factors2[0].Source() == "modified" {
		t.Error("Factors() does not return a defensive copy")
	}
}
