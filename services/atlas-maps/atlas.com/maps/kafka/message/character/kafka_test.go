package character

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

func TestStatusEvent_LoginBody_Serialization(t *testing.T) {
	event := StatusEvent[StatusEventLoginBody]{
		TransactionId: uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		CharacterId:   12345,
		Type:          EventCharacterStatusTypeLogin,
		WorldId:       world.Id(1),
		Body: StatusEventLoginBody{
			ChannelId: channel.Id(2),
			MapId:     _map.Id(100000000),
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var decoded StatusEvent[StatusEventLoginBody]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if decoded.TransactionId != event.TransactionId {
		t.Errorf("Expected TransactionId %v, got %v", event.TransactionId, decoded.TransactionId)
	}
	if decoded.CharacterId != event.CharacterId {
		t.Errorf("Expected CharacterId %d, got %d", event.CharacterId, decoded.CharacterId)
	}
	if decoded.Type != event.Type {
		t.Errorf("Expected Type %s, got %s", event.Type, decoded.Type)
	}
	if decoded.WorldId != event.WorldId {
		t.Errorf("Expected WorldId %d, got %d", event.WorldId, decoded.WorldId)
	}
	if decoded.Body.ChannelId != event.Body.ChannelId {
		t.Errorf("Expected ChannelId %d, got %d", event.Body.ChannelId, decoded.Body.ChannelId)
	}
	if decoded.Body.MapId != event.Body.MapId {
		t.Errorf("Expected MapId %d, got %d", event.Body.MapId, decoded.Body.MapId)
	}
}

func TestStatusEvent_LogoutBody_Serialization(t *testing.T) {
	event := StatusEvent[StatusEventLogoutBody]{
		TransactionId: uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		CharacterId:   12345,
		Type:          EventCharacterStatusTypeLogout,
		WorldId:       world.Id(1),
		Body: StatusEventLogoutBody{
			ChannelId: channel.Id(2),
			MapId:     _map.Id(100000000),
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var decoded StatusEvent[StatusEventLogoutBody]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if decoded.Type != EventCharacterStatusTypeLogout {
		t.Errorf("Expected Type %s, got %s", EventCharacterStatusTypeLogout, decoded.Type)
	}
	if decoded.Body.ChannelId != event.Body.ChannelId {
		t.Errorf("Expected ChannelId %d, got %d", event.Body.ChannelId, decoded.Body.ChannelId)
	}
	if decoded.Body.MapId != event.Body.MapId {
		t.Errorf("Expected MapId %d, got %d", event.Body.MapId, decoded.Body.MapId)
	}
}

func TestStatusEvent_MapChangedBody_Serialization(t *testing.T) {
	event := StatusEvent[StatusEventMapChangedBody]{
		TransactionId: uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		CharacterId:   12345,
		Type:          EventCharacterStatusTypeMapChanged,
		WorldId:       world.Id(1),
		Body: StatusEventMapChangedBody{
			ChannelId:      channel.Id(2),
			OldMapId:       _map.Id(100000000),
			TargetMapId:    _map.Id(100000001),
			TargetPortalId: 0,
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var decoded StatusEvent[StatusEventMapChangedBody]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if decoded.Type != EventCharacterStatusTypeMapChanged {
		t.Errorf("Expected Type %s, got %s", EventCharacterStatusTypeMapChanged, decoded.Type)
	}
	if decoded.Body.ChannelId != event.Body.ChannelId {
		t.Errorf("Expected ChannelId %d, got %d", event.Body.ChannelId, decoded.Body.ChannelId)
	}
	if decoded.Body.OldMapId != event.Body.OldMapId {
		t.Errorf("Expected OldMapId %d, got %d", event.Body.OldMapId, decoded.Body.OldMapId)
	}
	if decoded.Body.TargetMapId != event.Body.TargetMapId {
		t.Errorf("Expected TargetMapId %d, got %d", event.Body.TargetMapId, decoded.Body.TargetMapId)
	}
	if decoded.Body.TargetPortalId != event.Body.TargetPortalId {
		t.Errorf("Expected TargetPortalId %d, got %d", event.Body.TargetPortalId, decoded.Body.TargetPortalId)
	}
}

func TestStatusEvent_ChannelChangedBody_Serialization(t *testing.T) {
	event := StatusEvent[ChangeChannelEventLoginBody]{
		TransactionId: uuid.MustParse("12345678-1234-5678-1234-567812345678"),
		CharacterId:   12345,
		Type:          EventCharacterStatusTypeChannelChanged,
		WorldId:       world.Id(1),
		Body: ChangeChannelEventLoginBody{
			ChannelId:    channel.Id(2),
			OldChannelId: channel.Id(1),
			MapId:        _map.Id(100000000),
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var decoded StatusEvent[ChangeChannelEventLoginBody]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if decoded.Type != EventCharacterStatusTypeChannelChanged {
		t.Errorf("Expected Type %s, got %s", EventCharacterStatusTypeChannelChanged, decoded.Type)
	}
	if decoded.Body.ChannelId != event.Body.ChannelId {
		t.Errorf("Expected ChannelId %d, got %d", event.Body.ChannelId, decoded.Body.ChannelId)
	}
	if decoded.Body.OldChannelId != event.Body.OldChannelId {
		t.Errorf("Expected OldChannelId %d, got %d", event.Body.OldChannelId, decoded.Body.OldChannelId)
	}
	if decoded.Body.MapId != event.Body.MapId {
		t.Errorf("Expected MapId %d, got %d", event.Body.MapId, decoded.Body.MapId)
	}
}

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"EnvEventTopicCharacterStatus", EnvEventTopicCharacterStatus, "EVENT_TOPIC_CHARACTER_STATUS"},
		{"EventCharacterStatusTypeLogin", EventCharacterStatusTypeLogin, "LOGIN"},
		{"EventCharacterStatusTypeLogout", EventCharacterStatusTypeLogout, "LOGOUT"},
		{"EventCharacterStatusTypeChannelChanged", EventCharacterStatusTypeChannelChanged, "CHANNEL_CHANGED"},
		{"EventCharacterStatusTypeMapChanged", EventCharacterStatusTypeMapChanged, "MAP_CHANGED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %s to be '%s', got '%s'", tt.name, tt.expected, tt.value)
			}
		})
	}
}
