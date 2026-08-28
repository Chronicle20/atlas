package saga

import (
	"encoding/json"
	"strings"
	"testing"
)

// TemplateId is additive (task-125): absent in old payloads → 0; round-trips when set.
func TestDestroyAssetFromSlotPayloadTemplateIdRoundTrip(t *testing.T) {
	in := DestroyAssetFromSlotPayload{
		CharacterId:   1,
		InventoryType: 2,
		Slot:          3,
		Quantity:      1,
		TemplateId:    2290000,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out DestroyAssetFromSlotPayload
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.TemplateId != in.TemplateId {
		t.Errorf("templateId: got %d, want %d", out.TemplateId, in.TemplateId)
	}

	var legacy DestroyAssetFromSlotPayload
	if err := json.Unmarshal([]byte(`{"characterId":1,"inventoryType":2,"slot":3,"quantity":1,"showEffect":false}`), &legacy); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if legacy.TemplateId != 0 {
		t.Errorf("legacy templateId: got %d, want 0", legacy.TemplateId)
	}
}

func TestSkillBookUseSagaType(t *testing.T) {
	if string(SkillBookUse) != "skill_book_use" {
		t.Errorf("SkillBookUse: got %q", string(SkillBookUse))
	}
}

// AP and SP are additive (task-246 amendment 1): both omitempty, so a
// payload that omits them produces byte-identical JSON to today's
// producers, and a payload that sets them round-trips through JSON.
func TestCharacterCreatePayloadCarriesApAndSp(t *testing.T) {
	both := CharacterCreatePayload{AP: 12, SP: "3,0,0,0,0,0,0,0,0,0"}
	b, err := json.Marshal(both)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"ap":12`) {
		t.Errorf("expected \"ap\":12 in %s", s)
	}
	if !strings.Contains(s, `"sp":"3,0,0,0,0,0,0,0,0,0"`) {
		t.Errorf("expected sp field in %s", s)
	}

	var out CharacterCreatePayload
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.AP != 12 {
		t.Errorf("AP: got %d, want 12", out.AP)
	}
	if out.SP != "3,0,0,0,0,0,0,0,0,0" {
		t.Errorf("SP: got %q, want %q", out.SP, "3,0,0,0,0,0,0,0,0,0")
	}

	zero := CharacterCreatePayload{}
	zb, err := json.Marshal(zero)
	if err != nil {
		t.Fatalf("marshal zero: %v", err)
	}
	zs := string(zb)
	if strings.Contains(zs, `"ap"`) {
		t.Errorf("expected ap absent from omitempty zero value, got %s", zs)
	}
	if strings.Contains(zs, `"sp"`) {
		t.Errorf("expected sp absent from omitempty zero value, got %s", zs)
	}
}

func TestAwardCraftedAssetActionConstant(t *testing.T) {
	if string(AwardCraftedAsset) != "award_crafted_asset" {
		t.Errorf("AwardCraftedAsset: got %q", string(AwardCraftedAsset))
	}
}

func TestAwardCraftedAssetPayloadRoundTrip(t *testing.T) {
	in := AwardCraftedAssetPayload{
		CharacterId:   1,
		TemplateId:    1082002,
		Quantity:      1,
		Slots:         7,
		Strength:      3,
		Dexterity:     2,
		Intelligence:  0,
		Luck:          0,
		HP:            15,
		MP:            0,
		WeaponAttack:  4,
		MagicAttack:   0,
		WeaponDefense: 6,
		MagicDefense:  1,
		Accuracy:      2,
		Avoidability:  1,
		Hands:         0,
		Speed:         0,
		Jump:          0,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out AwardCraftedAssetPayload
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("round-trip mismatch: got %+v, want %+v", out, in)
	}
}

func TestAwardCraftedAssetPayloadSlotsSurvivesZero(t *testing.T) {
	in := AwardCraftedAssetPayload{
		CharacterId: 1,
		TemplateId:  1082002,
		Quantity:    1,
		Slots:       0,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"slots":0`) {
		t.Errorf("expected \"slots\":0 in %s", string(b))
	}
}

func TestAwardCraftedAssetStepUnmarshal(t *testing.T) {
	raw := `{"stepId":"award","status":"pending","action":"award_crafted_asset","payload":{"characterId":1,"templateId":1082002,"quantity":1,"slots":7,"strength":3,"weaponAttack":4}}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != AwardCraftedAsset {
		t.Errorf("Action: got %q, want %q", step.Action, AwardCraftedAsset)
	}
	payload, ok := step.Payload.(AwardCraftedAssetPayload)
	if !ok {
		t.Fatalf("Payload: got %T, want AwardCraftedAssetPayload", step.Payload)
	}
	if payload.TemplateId != 1082002 {
		t.Errorf("TemplateId: got %d, want 1082002", payload.TemplateId)
	}
	if payload.Slots != 7 {
		t.Errorf("Slots: got %d, want 7", payload.Slots)
	}
	if payload.Strength != 3 {
		t.Errorf("Strength: got %d, want 3", payload.Strength)
	}
	if payload.WeaponAttack != 4 {
		t.Errorf("WeaponAttack: got %d, want 4", payload.WeaponAttack)
	}
}
