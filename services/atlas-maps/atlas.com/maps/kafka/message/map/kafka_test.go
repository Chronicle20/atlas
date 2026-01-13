package _map

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

func TestStatusEvent_CharacterEnter_Serialization(t *testing.T) {
	event := StatusEvent[CharacterEnter]{
		TransactionId: uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		WorldId:       world.Id(1),
		ChannelId:     channel.Id(2),
		MapId:         _map.Id(100000000),
		Type:          EventTopicMapStatusTypeCharacterEnter,
		Body: CharacterEnter{
			CharacterId: 12345,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var decoded StatusEvent[CharacterEnter]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if decoded.TransactionId != event.TransactionId {
		t.Errorf("Expected TransactionId %v, got %v", event.TransactionId, decoded.TransactionId)
	}
	if decoded.WorldId != event.WorldId {
		t.Errorf("Expected WorldId %d, got %d", event.WorldId, decoded.WorldId)
	}
	if decoded.ChannelId != event.ChannelId {
		t.Errorf("Expected ChannelId %d, got %d", event.ChannelId, decoded.ChannelId)
	}
	if decoded.MapId != event.MapId {
		t.Errorf("Expected MapId %d, got %d", event.MapId, decoded.MapId)
	}
	if decoded.Type != event.Type {
		t.Errorf("Expected Type %s, got %s", event.Type, decoded.Type)
	}
	if decoded.Body.CharacterId != event.Body.CharacterId {
		t.Errorf("Expected CharacterId %d, got %d", event.Body.CharacterId, decoded.Body.CharacterId)
	}
}

func TestStatusEvent_CharacterExit_Serialization(t *testing.T) {
	event := StatusEvent[CharacterExit]{
		TransactionId: uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		WorldId:       world.Id(1),
		ChannelId:     channel.Id(2),
		MapId:         _map.Id(100000000),
		Type:          EventTopicMapStatusTypeCharacterExit,
		Body: CharacterExit{
			CharacterId: 12345,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var decoded StatusEvent[CharacterExit]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if decoded.Type != EventTopicMapStatusTypeCharacterExit {
		t.Errorf("Expected Type %s, got %s", EventTopicMapStatusTypeCharacterExit, decoded.Type)
	}
	if decoded.Body.CharacterId != event.Body.CharacterId {
		t.Errorf("Expected CharacterId %d, got %d", event.Body.CharacterId, decoded.Body.CharacterId)
	}
}

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"EnvEventTopicMapStatus", EnvEventTopicMapStatus, "EVENT_TOPIC_MAP_STATUS"},
		{"EventTopicMapStatusTypeCharacterEnter", EventTopicMapStatusTypeCharacterEnter, "CHARACTER_ENTER"},
		{"EventTopicMapStatusTypeCharacterExit", EventTopicMapStatusTypeCharacterExit, "CHARACTER_EXIT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %s to be '%s', got '%s'", tt.name, tt.expected, tt.value)
			}
		})
	}
}

func TestCharacterEnter_JSONFields(t *testing.T) {
	body := CharacterEnter{CharacterId: 12345}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, ok := m["characterId"]; !ok {
		t.Error("Expected 'characterId' field in JSON output")
	}
}

func TestCharacterExit_JSONFields(t *testing.T) {
	body := CharacterExit{CharacterId: 12345}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, ok := m["characterId"]; !ok {
		t.Error("Expected 'characterId' field in JSON output")
	}
}
