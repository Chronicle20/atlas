package saga

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TemplateId is additive (task-125): absent in old payloads → 0; round-trips when set.
func TestDestroyAssetFromSlotPayloadTemplateIdRoundTrip(t *testing.T) {
	tests := []struct {
		name           string
		in             *DestroyAssetFromSlotPayload
		rawJSON        string
		wantTemplateId uint32
	}{
		{
			name: "set value round-trips",
			in: &DestroyAssetFromSlotPayload{
				CharacterId:   1,
				InventoryType: 2,
				Slot:          3,
				Quantity:      1,
				TemplateId:    2290000,
			},
			wantTemplateId: 2290000,
		},
		{
			name:           "absent in legacy payload defaults to 0",
			rawJSON:        `{"characterId":1,"inventoryType":2,"slot":3,"quantity":1,"showEffect":false}`,
			wantTemplateId: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw []byte
			if tt.in != nil {
				b, err := json.Marshal(tt.in)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				raw = b
			} else {
				raw = []byte(tt.rawJSON)
			}

			var out DestroyAssetFromSlotPayload
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if out.TemplateId != tt.wantTemplateId {
				t.Errorf("templateId: got %d, want %d", out.TemplateId, tt.wantTemplateId)
			}
		})
	}
}

func TestSkillBookUseSagaType(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "SkillBookUse", want: "skill_book_use"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(SkillBookUse) != tt.want {
				t.Errorf("SkillBookUse: got %q", string(SkillBookUse))
			}
		})
	}
}

// AP and SP are additive (task-246 amendment 1): both omitempty, so a
// payload that omits them produces byte-identical JSON to today's
// producers, and a payload that sets them round-trips through JSON.
func TestCharacterCreatePayloadCarriesApAndSp(t *testing.T) {
	tests := []struct {
		name     string
		in       CharacterCreatePayload
		wantApSp bool
		wantAP   uint16
		wantSP   string
	}{
		{
			name:     "both set",
			in:       CharacterCreatePayload{AP: 12, SP: "3,0,0,0,0,0,0,0,0,0"},
			wantApSp: true,
			wantAP:   12,
			wantSP:   "3,0,0,0,0,0,0,0,0,0",
		},
		{
			name:     "zero value omitted",
			in:       CharacterCreatePayload{},
			wantApSp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			s := string(b)

			if tt.wantApSp {
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
				if out.AP != tt.wantAP {
					t.Errorf("AP: got %d, want %d", out.AP, tt.wantAP)
				}
				if out.SP != tt.wantSP {
					t.Errorf("SP: got %q, want %q", out.SP, tt.wantSP)
				}
			} else {
				if strings.Contains(s, `"ap"`) {
					t.Errorf("expected ap absent from omitempty zero value, got %s", s)
				}
				if strings.Contains(s, `"sp"`) {
					t.Errorf("expected sp absent from omitempty zero value, got %s", s)
				}
			}
		})
	}
}

// SpawnIfAbsent rides the spawn_monster step to atlas-monsters (task-290 A6):
// the decision of whether to suppress the spawn is made by atlas-monsters
// against its own registry, not here — this payload only carries the flag.
func TestSpawnMonsterPayloadSpawnIfAbsentOmitempty(t *testing.T) {
	tests := []struct {
		name          string
		spawnIfAbsent bool
	}{
		{name: "true round-trips", spawnIfAbsent: true},
		{name: "false omitted from JSON", spawnIfAbsent: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				SpawnIfAbsent: tt.spawnIfAbsent,
			}
			b, err := json.Marshal(in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			s := string(b)

			if tt.spawnIfAbsent {
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
			} else {
				if strings.Contains(s, "spawnIfAbsent") {
					t.Errorf("expected spawnIfAbsent absent from omitempty false value, got %s", s)
				}
			}
		})
	}
}
