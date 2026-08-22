package tenants

import (
	"atlas-configurations/tenants/characters"
	"atlas-configurations/tenants/npcs"
	"atlas-configurations/tenants/socket"
	"atlas-configurations/tenants/socket/handler"
	"atlas-configurations/tenants/socket/writer"
	"atlas-configurations/tenants/worlds"
	"encoding/json"
	"testing"
)

func TestTenantRestModelCarriesMapleLife(t *testing.T) {
	t.Run("document with mapleLife key", func(t *testing.T) {
		doc := `{
			"region": "GMS",
			"majorVersion": 83,
			"minorVersion": 1,
			"mapleLife": {
				"looks": [
					{"gender": 0, "faces": [20000], "hairs": [30000], "hairColors": [0], "skinColors": [0]}
				],
				"classes": [
					{"ordinal": 0, "gender": 0, "jobId": 100, "level": 10, "mapId": 10000, "stats": {"str": 35, "dex": 20, "int": 4, "luk": 4, "hp": 100, "mp": 50}, "ap": 5, "sp": "1,0,0,0,0,0,0,0,0,0", "spSkillId": 1000001, "meso": 1000, "equipment": [], "inventory": []}
				]
			}
		}`

		var decoded RestModel
		if err := json.Unmarshal([]byte(doc), &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if len(decoded.MapleLife.Looks) != 1 {
			t.Errorf("expected 1 look, got %d", len(decoded.MapleLife.Looks))
		}
		if len(decoded.MapleLife.Classes) != 1 {
			t.Errorf("expected 1 class, got %d", len(decoded.MapleLife.Classes))
		}
		if decoded.Region != "GMS" {
			t.Errorf("expected Region 'GMS', got '%s'", decoded.Region)
		}
	})

	t.Run("document without mapleLife key", func(t *testing.T) {
		doc := `{"region": "GMS", "majorVersion": 83, "minorVersion": 1}`

		var decoded RestModel
		if err := json.Unmarshal([]byte(doc), &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if len(decoded.MapleLife.Looks) != 0 {
			t.Errorf("expected 0 looks, got %d", len(decoded.MapleLife.Looks))
		}
		if len(decoded.MapleLife.Classes) != 0 {
			t.Errorf("expected 0 classes, got %d", len(decoded.MapleLife.Classes))
		}
		if decoded.Region != "GMS" {
			t.Errorf("expected Region 'GMS', got '%s'", decoded.Region)
		}
	})
}

func TestRestModel_GetName(t *testing.T) {
	rm := RestModel{}
	expected := "tenants"
	if rm.GetName() != expected {
		t.Errorf("expected GetName() to return '%s', got '%s'", expected, rm.GetName())
	}
}

func TestRestModel_GetID(t *testing.T) {
	testID := "test-uuid-123"
	rm := RestModel{Id: testID}

	if rm.GetID() != testID {
		t.Errorf("expected GetID() to return '%s', got '%s'", testID, rm.GetID())
	}
}

func TestRestModel_SetID(t *testing.T) {
	rm := RestModel{}
	testID := "new-test-id"

	err := rm.SetID(testID)
	if err != nil {
		t.Fatalf("SetID returned error: %v", err)
	}

	if rm.Id != testID {
		t.Errorf("expected Id to be '%s', got '%s'", testID, rm.Id)
	}

	if rm.GetID() != testID {
		t.Errorf("expected GetID() to return '%s', got '%s'", testID, rm.GetID())
	}
}

func TestRestModel_JSONMarshal(t *testing.T) {
	rm := RestModel{
		Id:           "test-id",
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
		UsesPin:      true,
		Socket: socket.RestModel{
			Handlers: []handler.RestModel{},
			Writers:  []writer.RestModel{},
		},
		Characters: characters.RestModel{},
		NPCs:       []npcs.RestModel{},
		Worlds:     []worlds.RestModel{},
	}

	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded RestModel
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Id should not be marshaled (json:"-")
	if decoded.Id != "" {
		t.Errorf("expected Id to be empty after unmarshal, got '%s'", decoded.Id)
	}

	if decoded.Region != rm.Region {
		t.Errorf("expected Region '%s', got '%s'", rm.Region, decoded.Region)
	}
	if decoded.MajorVersion != rm.MajorVersion {
		t.Errorf("expected MajorVersion %d, got %d", rm.MajorVersion, decoded.MajorVersion)
	}
	if decoded.MinorVersion != rm.MinorVersion {
		t.Errorf("expected MinorVersion %d, got %d", rm.MinorVersion, decoded.MinorVersion)
	}
	if decoded.UsesPin != rm.UsesPin {
		t.Errorf("expected UsesPin %v, got %v", rm.UsesPin, decoded.UsesPin)
	}
}

func TestRestModel_JSONMarshalWithNestedData(t *testing.T) {
	rm := RestModel{
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
		UsesPin:      true,
		Socket: socket.RestModel{
			Handlers: []handler.RestModel{
				{OpCode: "0x01", Validator: "default", Handler: "handler1"},
			},
			Writers: []writer.RestModel{
				{OpCode: "0x64", Writer: "writer1"},
			},
		},
		NPCs: []npcs.RestModel{
			{NPCId: 1000, Impl: "npc1"},
		},
		Worlds: []worlds.RestModel{
			{Name: "Scania", Flag: "0", ServerMessage: "Welcome"},
		},
	}

	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded RestModel
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if len(decoded.Socket.Handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(decoded.Socket.Handlers))
	}
	if len(decoded.Socket.Writers) != 1 {
		t.Errorf("expected 1 writer, got %d", len(decoded.Socket.Writers))
	}
	if len(decoded.NPCs) != 1 {
		t.Errorf("expected 1 NPC, got %d", len(decoded.NPCs))
	}
	if len(decoded.Worlds) != 1 {
		t.Errorf("expected 1 world, got %d", len(decoded.Worlds))
	}
}

func TestRestModel_EmptyState(t *testing.T) {
	rm := RestModel{}

	if rm.GetName() != "tenants" {
		t.Errorf("expected GetName() to return 'tenants' for empty model")
	}

	if rm.GetID() != "" {
		t.Errorf("expected GetID() to return empty string for empty model, got '%s'", rm.GetID())
	}
}
