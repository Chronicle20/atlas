package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestEffectSkillAffectedForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewEffectSkillAffectedForeign(123, 2, 1001, 5)
	output := EffectSkillAffectedForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.Mode() != input.Mode() {
		t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
	}
	if output.SkillId() != input.SkillId() {
		t.Errorf("skillId: got %v, want %v", output.SkillId(), input.SkillId())
	}
	if output.SkillLevel() != input.SkillLevel() {
		t.Errorf("skillLevel: got %v, want %v", output.SkillLevel(), input.SkillLevel())
	}
}

func TestEffectPetForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewEffectPetForeign(456, 4, 1, 2)
	output := EffectPetForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.Mode() != input.Mode() {
		t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
	}
	if output.EffectType() != input.EffectType() {
		t.Errorf("effectType: got %v, want %v", output.EffectType(), input.EffectType())
	}
	if output.PetIndex() != input.PetIndex() {
		t.Errorf("petIndex: got %v, want %v", output.PetIndex(), input.PetIndex())
	}
}

func TestEffectWithIdForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewEffectWithIdForeign(789, 5, 2000100)
	output := EffectWithIdForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.Id() != input.Id() {
		t.Errorf("id: got %v, want %v", output.Id(), input.Id())
	}
}

func TestEffectWithMessageForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewEffectWithMessageForeign(100, 12, "Effect/BasicEff/Inku")
	output := EffectWithMessageForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.Message() != input.Message() {
		t.Errorf("message: got %v, want %v", output.Message(), input.Message())
	}
}

func TestEffectProtectOnDieForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	// Test with safetyCharm=true (no itemId)
	input := NewEffectProtectOnDieForeign(200, 6, true, 3, 7, 0)
	output := EffectProtectOnDieForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.SafetyCharm() != input.SafetyCharm() {
		t.Errorf("safetyCharm: got %v, want %v", output.SafetyCharm(), input.SafetyCharm())
	}

	// Test with safetyCharm=false (has itemId)
	input2 := NewEffectProtectOnDieForeign(200, 6, false, 2, 5, 2100000)
	output2 := EffectProtectOnDieForeign{}
	pt.RoundTrip(t, ctx, input2.Encode, output2.Decode, nil)
	if output2.ItemId() != input2.ItemId() {
		t.Errorf("itemId: got %v, want %v", output2.ItemId(), input2.ItemId())
	}
}

func TestEffectIncDecHPForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewEffectIncDecHPForeign(300, 10, -5)
	output := EffectIncDecHPForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.Delta() != input.Delta() {
		t.Errorf("delta: got %v, want %v", output.Delta(), input.Delta())
	}
}

func TestEffectShowInfoForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewEffectShowInfoForeign(400, 23, "Map/Effect/quest")
	output := EffectShowInfoForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.Path() != input.Path() {
		t.Errorf("path: got %v, want %v", output.Path(), input.Path())
	}
}

func TestEffectLotteryUseForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	// success=true (has message)
	input := NewEffectLotteryUseForeign(500, 14, 5220000, true, "You won!")
	output := EffectLotteryUseForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.Message() != input.Message() {
		t.Errorf("message: got %v, want %v", output.Message(), input.Message())
	}

	// success=false (no message)
	input2 := NewEffectLotteryUseForeign(500, 14, 5220000, false, "")
	output2 := EffectLotteryUseForeign{}
	pt.RoundTrip(t, ctx, input2.Encode, output2.Decode, nil)
	if output2.Success() != false {
		t.Errorf("success: got %v, want false", output2.Success())
	}
}

func TestEffectItemMakerForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewEffectItemMakerForeign(600, 16, 1)
	output := EffectItemMakerForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.State() != input.State() {
		t.Errorf("state: got %v, want %v", output.State(), input.State())
	}
}

func TestEffectUpgradeTombForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewEffectUpgradeTombForeign(700, 21, 5)
	output := EffectUpgradeTombForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.UsesRemaining() != input.UsesRemaining() {
		t.Errorf("usesRemaining: got %v, want %v", output.UsesRemaining(), input.UsesRemaining())
	}
}

func TestEffectIncubatorUseForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewEffectIncubatorUseForeign(800, 24, 5000028, "Effect/BasicEff")
	output := EffectIncubatorUseForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if output.ItemId() != input.ItemId() {
		t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
	}
	if output.Message() != input.Message() {
		t.Errorf("message: got %v, want %v", output.Message(), input.Message())
	}
}

func TestEffectQuestForeignRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	// With rewards
	rewards := []QuestReward{{ItemId: 4001000, Amount: 5}, {ItemId: 4001001, Amount: -2}}
	input := NewEffectQuestForeign(900, 3, "", 0, rewards)
	output := EffectQuestForeign{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.CharacterId() != input.CharacterId() {
		t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
	}
	if len(output.Rewards()) != len(input.Rewards()) {
		t.Errorf("rewards count: got %v, want %v", len(output.Rewards()), len(input.Rewards()))
	}

	// Without rewards (message path)
	input2 := NewEffectQuestForeign(900, 3, "Quest complete!", 42, nil)
	output2 := EffectQuestForeign{}
	pt.RoundTrip(t, ctx, input2.Encode, output2.Decode, nil)
	if output2.Message() != input2.Message() {
		t.Errorf("message: got %v, want %v", output2.Message(), input2.Message())
	}
	if output2.NEffect() != input2.NEffect() {
		t.Errorf("nEffect: got %v, want %v", output2.NEffect(), input2.NEffect())
	}
}

func TestEffectQuestRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	// With rewards
	rewards := []QuestReward{{ItemId: 4001000, Amount: 5}}
	input := NewEffectQuest(3, "", 0, rewards)
	output := EffectQuest{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if len(output.Rewards()) != 1 {
		t.Errorf("rewards count: got %v, want 1", len(output.Rewards()))
	}

	// Without rewards
	input2 := NewEffectQuest(3, "Done!", 10, nil)
	output2 := EffectQuest{}
	pt.RoundTrip(t, ctx, input2.Encode, output2.Decode, nil)
	if output2.Message() != "Done!" {
		t.Errorf("message: got %v, want Done!", output2.Message())
	}
}
