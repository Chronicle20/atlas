package character

import (
	"encoding/binary"
	"encoding/json"
	"testing"
)

// extractKeyAsUint32 extracts the characterId from a binary-encoded key
func extractKeyAsUint32(key []byte) uint32 {
	if len(key) < 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(key)
}

func TestEnableActionsProvider_MessageStructure(t *testing.T) {
	worldId := byte(1)
	channelId := byte(2)
	characterId := uint32(12345)

	provider := enableActionsProvider(worldId, channelId, characterId)
	messages, err := provider()

	if err != nil {
		t.Fatalf("enableActionsProvider() returned unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	msg := messages[0]

	// Verify key is based on characterId (binary encoded little-endian)
	keyValue := extractKeyAsUint32(msg.Key)
	if keyValue != characterId {
		t.Errorf("message key = %d, want %d", keyValue, characterId)
	}

	// Verify message value structure
	var event statusEvent[statusEventStatChangedBody]
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		t.Fatalf("failed to unmarshal message value: %v", err)
	}

	if event.CharacterId != characterId {
		t.Errorf("CharacterId = %d, want %d", event.CharacterId, characterId)
	}

	if event.Type != EventCharacterStatusTypeStatChanged {
		t.Errorf("Type = %s, want %s", event.Type, EventCharacterStatusTypeStatChanged)
	}

	if event.WorldId != worldId {
		t.Errorf("WorldId = %d, want %d", event.WorldId, worldId)
	}

	if event.Body.ChannelId != channelId {
		t.Errorf("Body.ChannelId = %d, want %d", event.Body.ChannelId, channelId)
	}

	if event.Body.ExclRequestSent != true {
		t.Errorf("Body.ExclRequestSent = %v, want true", event.Body.ExclRequestSent)
	}
}

func TestChangeMapProvider_MessageStructure(t *testing.T) {
	worldId := byte(1)
	channelId := byte(2)
	characterId := uint32(67890)
	mapId := uint32(100000000)
	portalId := uint32(5)

	provider := ChangeMapProvider(worldId, channelId, characterId, mapId, portalId)
	messages, err := provider()

	if err != nil {
		t.Fatalf("ChangeMapProvider() returned unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	msg := messages[0]

	// Verify key is based on characterId (binary encoded little-endian)
	keyValue := extractKeyAsUint32(msg.Key)
	if keyValue != characterId {
		t.Errorf("message key = %d, want %d", keyValue, characterId)
	}

	// Verify message value structure
	var event commandEvent[changeMapBody]
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		t.Fatalf("failed to unmarshal message value: %v", err)
	}

	if event.CharacterId != characterId {
		t.Errorf("CharacterId = %d, want %d", event.CharacterId, characterId)
	}

	if event.Type != CommandCharacterChangeMap {
		t.Errorf("Type = %s, want %s", event.Type, CommandCharacterChangeMap)
	}

	if event.WorldId != worldId {
		t.Errorf("WorldId = %d, want %d", event.WorldId, worldId)
	}

	if event.Body.ChannelId != channelId {
		t.Errorf("Body.ChannelId = %d, want %d", event.Body.ChannelId, channelId)
	}

	if event.Body.MapId != mapId {
		t.Errorf("Body.MapId = %d, want %d", event.Body.MapId, mapId)
	}

	if event.Body.PortalId != portalId {
		t.Errorf("Body.PortalId = %d, want %d", event.Body.PortalId, portalId)
	}
}

func TestEnableActionsProvider_DifferentCharacterIds(t *testing.T) {
	tests := []struct {
		name        string
		characterId uint32
	}{
		{"small id", 1},
		{"medium id", 12345},
		{"large id", 4294967295},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := enableActionsProvider(1, 1, tt.characterId)
			messages, err := provider()

			if err != nil {
				t.Fatalf("enableActionsProvider() returned unexpected error: %v", err)
			}

			keyValue := extractKeyAsUint32(messages[0].Key)
			if keyValue != tt.characterId {
				t.Errorf("message key = %d, want %d", keyValue, tt.characterId)
			}
		})
	}
}

func TestChangeMapProvider_DifferentMapIds(t *testing.T) {
	tests := []struct {
		name   string
		mapId  uint32
		portal uint32
	}{
		{"henesys", 100000000, 0},
		{"ellinia", 101000000, 5},
		{"perion", 102000000, 10},
		{"kerning", 103000000, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := ChangeMapProvider(1, 1, 12345, tt.mapId, tt.portal)
			messages, err := provider()

			if err != nil {
				t.Fatalf("ChangeMapProvider() returned unexpected error: %v", err)
			}

			var event commandEvent[changeMapBody]
			if err := json.Unmarshal(messages[0].Value, &event); err != nil {
				t.Fatalf("failed to unmarshal message value: %v", err)
			}

			if event.Body.MapId != tt.mapId {
				t.Errorf("Body.MapId = %d, want %d", event.Body.MapId, tt.mapId)
			}

			if event.Body.PortalId != tt.portal {
				t.Errorf("Body.PortalId = %d, want %d", event.Body.PortalId, tt.portal)
			}
		})
	}
}
