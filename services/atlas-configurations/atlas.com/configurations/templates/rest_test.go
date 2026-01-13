package templates

import (
	"encoding/json"
	"testing"

	"atlas-configurations/templates/characters"
	"atlas-configurations/templates/npcs"
	"atlas-configurations/templates/socket"
	"atlas-configurations/templates/socket/handler"
	"atlas-configurations/templates/socket/writer"
	"atlas-configurations/templates/worlds"
)

func TestRestModel_GetName(t *testing.T) {
	rm := RestModel{}
	expected := "templates"
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
			{Name: "Scania", Flag: "0"},
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

	if rm.GetName() != "templates" {
		t.Errorf("expected GetName() to return 'templates' for empty model")
	}

	if rm.GetID() != "" {
		t.Errorf("expected GetID() to return empty string for empty model, got '%s'", rm.GetID())
	}
}
