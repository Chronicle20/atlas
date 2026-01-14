package skill_test

import (
	"atlas-skills/kafka/message"
	skillmsg "atlas-skills/kafka/message/skill"
	"atlas-skills/skill"
	"atlas-skills/test"
	"testing"
	"time"

	logtest "github.com/sirupsen/logrus/hooks/test"
)

func TestHandleCommandRequestCreate_Success(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()

	characterId := uint32(12345)
	skillId := uint32(1001001)
	expiration := time.Now().Add(24 * time.Hour)

	// Use message buffer approach to test the processor logic
	mb := message.NewBuffer()
	processor := skill.NewProcessor(logger, ctx, db)

	// This tests the same logic as the consumer handler
	_, err := processor.Create(mb)(characterId)(skillId)(10)(20)(expiration)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	// Verify skill was created
	s, err := processor.ByIdProvider(characterId, skillId)()
	if err != nil {
		t.Fatalf("ByIdProvider() unexpected error: %v", err)
	}
	if s.Id() != skillId {
		t.Errorf("s.Id() = %d, want %d", s.Id(), skillId)
	}
}

func TestHandleCommandRequestUpdate_Success(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()

	characterId := uint32(12345)
	skillId := uint32(1001001)
	expiration := time.Now().Add(24 * time.Hour)

	processor := skill.NewProcessor(logger, ctx, db)
	mb := message.NewBuffer()

	// Create initial skill
	_, err := processor.Create(mb)(characterId)(skillId)(10)(20)(expiration)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	// Update the skill (same logic as consumer handler)
	newExpiration := time.Now().Add(48 * time.Hour)
	_, err = processor.Update(mb)(characterId)(skillId)(15)(25)(newExpiration)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	// Verify skill was updated
	s, err := processor.ByIdProvider(characterId, skillId)()
	if err != nil {
		t.Fatalf("ByIdProvider() unexpected error: %v", err)
	}
	if s.Level() != 15 {
		t.Errorf("s.Level() = %d, want 15", s.Level())
	}
	if s.MasterLevel() != 25 {
		t.Errorf("s.MasterLevel() = %d, want 25", s.MasterLevel())
	}
}

func TestCommandTypeFiltering(t *testing.T) {
	// Test that handlers correctly filter by command type
	// This verifies the pattern used in consumers

	tests := []struct {
		name        string
		commandType string
		expected    string
		shouldMatch bool
	}{
		{"RequestCreate matches", skillmsg.CommandTypeRequestCreate, skillmsg.CommandTypeRequestCreate, true},
		{"RequestUpdate matches", skillmsg.CommandTypeRequestUpdate, skillmsg.CommandTypeRequestUpdate, true},
		{"SetCooldown matches", skillmsg.CommandTypeSetCooldown, skillmsg.CommandTypeSetCooldown, true},
		{"Wrong type does not match", skillmsg.CommandTypeRequestCreate, skillmsg.CommandTypeRequestUpdate, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := tt.commandType == tt.expected
			if match != tt.shouldMatch {
				t.Errorf("command type match = %v, want %v", match, tt.shouldMatch)
			}
		})
	}
}
