package saga

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
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

// SpawnIfAbsent rides the spawn_monster step to atlas-monsters (task-290 A6):
// the decision of whether to suppress the spawn is made by atlas-monsters
// against its own registry, not here — this payload only carries the flag.
func TestSpawnMonsterPayloadRoundTripsSpawnIfAbsent(t *testing.T) {
	in := SpawnMonsterPayload{
		CharacterId:   1,
		WorldId:       0,
		ChannelId:     1,
		MapId:         926000000,
		Instance:      uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		MonsterId:     9100013,
		X:             82,
		Y:             200,
		Count:         1,
		SpawnIfAbsent: true,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"spawnIfAbsent":true`) {
		t.Errorf("expected \"spawnIfAbsent\":true in %s", s)
	}

	var out SpawnMonsterPayload
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.SpawnIfAbsent != true {
		t.Errorf("SpawnIfAbsent: got %v, want true", out.SpawnIfAbsent)
	}
}

func TestSpawnMonsterPayloadOmitsSpawnIfAbsentWhenFalse(t *testing.T) {
	in := SpawnMonsterPayload{
		CharacterId:   1,
		WorldId:       0,
		ChannelId:     1,
		MapId:         926000000,
		Instance:      uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		MonsterId:     9100013,
		X:             82,
		Y:             200,
		Count:         1,
		SpawnIfAbsent: false,
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if strings.Contains(s, "spawnIfAbsent") {
		t.Errorf("expected spawnIfAbsent absent from omitempty false value, got %s", s)
	}
}
