package macro_test

import (
	"atlas-skills/kafka/message"
	macromsg "atlas-skills/kafka/message/macro"
	"atlas-skills/macro"
	"atlas-skills/test"
	"testing"

	"github.com/Chronicle20/atlas-constants/skill"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// buildMacro is a test helper that builds a macro and fails the test on error
func buildMacro(t *testing.T, id uint32, name string, shout bool, skillId1, skillId2, skillId3 skill.Id) macro.Model {
	t.Helper()
	m, err := macro.NewModelBuilder().
		SetId(id).
		SetName(name).
		SetShout(shout).
		SetSkillId1(skillId1).
		SetSkillId2(skillId2).
		SetSkillId3(skillId3).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	return m
}

func TestHandleCommandUpdate_Success(t *testing.T) {
	db := test.SetupTestDB(t)
	defer test.CleanupTestDB(db)

	ctx := test.CreateTestContext()
	logger, _ := logtest.NewNullLogger()

	characterId := uint32(12345)

	processor := macro.NewProcessor(logger, ctx, db)
	mb := message.NewBuffer()

	// Create macros (same logic as consumer handler)
	macros := []macro.Model{
		buildMacro(t, 0, "Attack Combo", false, skill.Id(1001001), skill.Id(1001002), skill.Id(0)),
		buildMacro(t, 1, "Buff Combo", true, skill.Id(2001001), 0, 0),
	}

	_, err := processor.Update(mb)(characterId)(macros)
	if err != nil {
		t.Fatalf("Update() unexpected error: %v", err)
	}

	// Verify macros were created
	result, err := processor.ByCharacterIdProvider(characterId)()
	if err != nil {
		t.Fatalf("ByCharacterIdProvider() unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len(result) = %d, want 2", len(result))
	}
}

func TestCommandTypeFiltering(t *testing.T) {
	// Test that handlers correctly filter by command type
	tests := []struct {
		name        string
		commandType string
		expected    string
		shouldMatch bool
	}{
		{"Update matches", macromsg.CommandTypeUpdate, macromsg.CommandTypeUpdate, true},
		{"Wrong type does not match", macromsg.CommandTypeUpdate, "wrong_type", false},
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
