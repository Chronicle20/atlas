package monster

import (
	"testing"
)

func TestDamageDistributionBuilder_Build_Success(t *testing.T) {
	m, err := NewDamageDistributionBuilder().Build()
	if err != nil {
		t.Errorf("Expected no error for default build, got %v", err)
	}
	if m.Solo() == nil {
		t.Errorf("Expected solo map to be initialized")
	}
	if m.PersonalRatio() == nil {
		t.Errorf("Expected personalRatio map to be initialized")
	}
}

func TestDamageDistributionBuilder_NilSoloMap(t *testing.T) {
	b := &DamageDistributionBuilder{
		solo:          nil,
		party:         make(map[uint32]map[uint32]uint32),
		personalRatio: make(map[uint32]float64),
	}
	_, err := b.Build()
	if err == nil {
		t.Errorf("Expected error when solo map is nil")
	}
	expectedMsg := "solo map cannot be nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestDamageDistributionBuilder_NilPartyMap(t *testing.T) {
	b := &DamageDistributionBuilder{
		solo:          make(map[uint32]uint32),
		party:         nil,
		personalRatio: make(map[uint32]float64),
	}
	_, err := b.Build()
	if err == nil {
		t.Errorf("Expected error when party map is nil")
	}
	expectedMsg := "party map cannot be nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestDamageDistributionBuilder_NilPersonalRatioMap(t *testing.T) {
	b := &DamageDistributionBuilder{
		solo:          make(map[uint32]uint32),
		party:         make(map[uint32]map[uint32]uint32),
		personalRatio: nil,
	}
	_, err := b.Build()
	if err == nil {
		t.Errorf("Expected error when personalRatio map is nil")
	}
	expectedMsg := "personalRatio map cannot be nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestDamageDistributionBuilder_WithValues(t *testing.T) {
	solo := map[uint32]uint32{1: 100}
	party := map[uint32]map[uint32]uint32{1: {2: 50}}
	personalRatio := map[uint32]float64{1: 0.5}

	m, err := NewDamageDistributionBuilder().
		SetSolo(solo).
		SetParty(party).
		SetPersonalRatio(personalRatio).
		SetExperiencePerDamage(2.5).
		SetStandardDeviationRatio(0.3).
		Build()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if m.Solo()[1] != 100 {
		t.Errorf("Expected solo[1]=100, got %d", m.Solo()[1])
	}
	if m.ExperiencePerDamage() != 2.5 {
		t.Errorf("Expected experiencePerDamage=2.5, got %f", m.ExperiencePerDamage())
	}
}
