package family

import (
	"testing"

	world2 "github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

func TestFamilyMember_Accessors(t *testing.T) {
	// Create a test family member
	characterId := uint32(12345)
	tenantId := uuid.New()
	level := uint16(50)
	world := world2.Id(1)
	seniorId := uint32(99999)
	juniorIds := []uint32{11111, 22222}
	rep := uint32(1000)
	dailyRep := uint32(100)

	member, err := NewBuilder(characterId, tenantId, level, world).
		SetSeniorId(seniorId).
		SetJuniorIds(juniorIds).
		SetRep(rep).
		SetDailyRep(dailyRep).
		Build()

	if err != nil {
		t.Fatalf("Failed to build family member: %v", err)
	}

	// Test accessors
	if member.CharacterId() != characterId {
		t.Errorf("Expected CharacterId %d, got %d", characterId, member.CharacterId())
	}

	if member.TenantId() != tenantId {
		t.Errorf("Expected TenantId %v, got %v", tenantId, member.TenantId())
	}

	if member.Level() != level {
		t.Errorf("Expected Level %d, got %d", level, member.Level())
	}

	if member.World() != world {
		t.Errorf("Expected World %d, got %d", world, member.World())
	}

	if member.SeniorId() == nil || *member.SeniorId() != seniorId {
		actualSeniorId := uint32(0)
		if member.SeniorId() != nil {
			actualSeniorId = *member.SeniorId()
		}
		t.Errorf("Expected SeniorId %d, got %d", seniorId, actualSeniorId)
	}

	if len(member.JuniorIds()) != 2 {
		t.Errorf("Expected 2 juniors, got %d", len(member.JuniorIds()))
	}

	if member.Rep() != rep {
		t.Errorf("Expected Rep %d, got %d", rep, member.Rep())
	}

	if member.DailyRep() != dailyRep {
		t.Errorf("Expected DailyRep %d, got %d", dailyRep, member.DailyRep())
	}
}

func TestFamilyMember_BusinessLogic(t *testing.T) {
	characterId := uint32(12345)
	tenantId := uuid.New()
	level := uint16(50)
	world := world2.Id(1)

	member, err := NewBuilder(characterId, tenantId, level, world).Build()
	if err != nil {
		t.Fatalf("Failed to build family member: %v", err)
	}

	// Test initial state
	if member.HasSenior() {
		t.Error("New member should not have a senior")
	}

	if member.HasJuniors() {
		t.Error("New member should not have juniors")
	}

	if !member.CanAddJunior() {
		t.Error("New member should be able to add juniors")
	}

	// Test with juniors
	juniors := []uint32{11111, 22222}
	memberWithJuniors, err := NewBuilder(characterId, tenantId, level, world).
		SetJuniorIds(juniors).
		Build()
	if err != nil {
		t.Fatalf("Failed to build member with juniors: %v", err)
	}

	if !memberWithJuniors.HasJuniors() {
		t.Error("Member with juniors should report HasJuniors() as true")
	}

	if memberWithJuniors.CanAddJunior() {
		t.Error("Member with 2 juniors should not be able to add more")
	}

	// Test with senior
	seniorId := uint32(99999)
	memberWithSenior, err := NewBuilder(characterId, tenantId, level, world).
		SetSeniorId(seniorId).
		Build()
	if err != nil {
		t.Fatalf("Failed to build member with senior: %v", err)
	}

	if !memberWithSenior.HasSenior() {
		t.Error("Member with senior should report HasSenior() as true")
	}
}

func TestValidateLevelDifference(t *testing.T) {
	tests := []struct {
		name           string
		seniorLevel    uint16
		juniorLevel    uint16
		expectedResult bool
	}{
		{"Same level", 50, 50, true},
		{"Junior 1 level lower", 50, 49, true},
		{"Junior 20 levels lower", 50, 30, true},
		{"Junior 21 levels lower", 50, 29, false},
		{"Senior 1 level lower", 49, 50, true},
		{"Senior 20 levels lower", 30, 50, true},
		{"Senior 21 levels lower", 29, 50, false},
		{"Large difference", 100, 50, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateLevelDifference(tt.seniorLevel, tt.juniorLevel)
			if result != tt.expectedResult {
				t.Errorf("ValidateLevelDifference(%d, %d) = %v, want %v",
					tt.seniorLevel, tt.juniorLevel, result, tt.expectedResult)
			}
		})
	}
}

func TestValidateLocation(t *testing.T) {
	tests := []struct {
		name           string
		seniorWorld    byte
		seniorMap      uint32
		juniorWorld    byte
		juniorMap      uint32
		expectedResult bool
	}{
		{"Same location", 1, 100, 1, 100, true},
		{"Different world", 1, 100, 2, 100, false},
		{"Different map", 1, 100, 1, 200, false},
		{"Both different", 1, 100, 2, 200, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateLocation(tt.seniorWorld, tt.seniorMap, tt.juniorWorld, tt.juniorMap)
			if result != tt.expectedResult {
				t.Errorf("ValidateLocation(%d, %d, %d, %d) = %v, want %v",
					tt.seniorWorld, tt.seniorMap, tt.juniorWorld, tt.juniorMap, result, tt.expectedResult)
			}
		})
	}
}

func TestValidateDailyRepCap(t *testing.T) {
	tests := []struct {
		name            string
		currentDailyRep uint32
		additionalRep   uint32
		expectedResult  bool
	}{
		{"Under cap", 1000, 500, true},
		{"Exactly at cap", 5000, 0, true},
		{"Would exceed cap", 4500, 600, false},
		{"Already over cap", 5500, 100, false},
		{"Zero current, valid addition", 0, 1000, true},
		{"Zero current, exceeds cap", 0, 6000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateDailyRepCap(tt.currentDailyRep, tt.additionalRep)
			if result != tt.expectedResult {
				t.Errorf("ValidateDailyRepCap(%d, %d) = %v, want %v",
					tt.currentDailyRep, tt.additionalRep, result, tt.expectedResult)
			}
		})
	}
}

func TestFamilyMember_Immutability(t *testing.T) {
	characterId := uint32(12345)
	tenantId := uuid.New()
	level := uint16(50)
	world := world2.Id(1)
	originalJuniors := []uint32{11111, 22222}

	member, err := NewBuilder(characterId, tenantId, level, world).
		SetJuniorIds(originalJuniors).
		Build()
	if err != nil {
		t.Fatalf("Failed to build family member: %v", err)
	}

	// Get junior slice and modify it
	juniors := member.JuniorIds()
	if len(juniors) != 2 {
		t.Fatalf("Expected 2 juniors, got %d", len(juniors))
	}

	// Modify the returned slice
	juniors[0] = 99999

	// Verify original member is not affected
	originalJuniorsFromMember := member.JuniorIds()
	if originalJuniorsFromMember[0] == 99999 {
		t.Error("Family member should be immutable - external modification affected internal state")
	}

	if originalJuniorsFromMember[0] != 11111 {
		t.Errorf("Expected first junior to remain %d, got %d", 11111, originalJuniorsFromMember[0])
	}
}

func TestBuilder_Validation(t *testing.T) {
	characterId := uint32(12345)
	tenantId := uuid.New()
	level := uint16(50)
	world := world2.Id(1)

	t.Run("Valid builder", func(t *testing.T) {
		_, err := NewBuilder(characterId, tenantId, level, world).Build()
		if err != nil {
			t.Errorf("Valid builder should not return error: %v", err)
		}
	})

	t.Run("Too many juniors", func(t *testing.T) {
		tooManyJuniors := []uint32{11111, 22222, 33333}
		_, err := NewBuilder(characterId, tenantId, level, world).
			SetJuniorIds(tooManyJuniors).
			Build()
		if err == nil {
			t.Error("Builder should return error for more than 2 juniors")
		}
	})

	t.Run("Invalid daily rep", func(t *testing.T) {
		_, err := NewBuilder(characterId, tenantId, level, world).
			SetDailyRep(6000). // Over the cap
			Build()
		if err == nil {
			t.Error("Builder should return error for daily rep over 5000")
		}
	})

	t.Run("Self-reference in juniors", func(t *testing.T) {
		selfJuniors := []uint32{characterId, 22222}
		_, err := NewBuilder(characterId, tenantId, level, world).
			SetJuniorIds(selfJuniors).
			Build()
		if err == nil {
			t.Error("Builder should return error for self-reference in juniors")
		}
	})

	t.Run("Senior same as character", func(t *testing.T) {
		_, err := NewBuilder(characterId, tenantId, level, world).
			SetSeniorId(characterId).
			Build()
		if err == nil {
			t.Error("Builder should return error for self-reference as senior")
		}
	})
}
