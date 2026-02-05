package character

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

// extractKeyAsUint32 extracts the characterId from a binary-encoded key
func extractKeyAsUint32(key []byte) uint32 {
	if len(key) < 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(key)
}

func TestEnableActionsProvider_MessageStructure(t *testing.T) {
	worldId := world.Id(1)
	channelId := channel.Id(2)
	mapId := _map.Id(100000000)
	instance := uuid.New()
	characterId := uint32(12345)

	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	provider := enableActionsProvider(f, characterId)
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
	worldId := world.Id(1)
	channelId := channel.Id(2)
	mapId := _map.Id(100000000)
	instance := uuid.New()
	characterId := uint32(67890)
	targetMapId := _map.Id(200000000)
	portalId := uint32(5)

	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()

	provider := ChangeMapProvider(f, characterId, targetMapId, portalId)
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

	if event.Body.MapId != targetMapId {
		t.Errorf("Body.MapId = %d, want %d", event.Body.MapId, targetMapId)
	}

	if event.Body.PortalId != portalId {
		t.Errorf("Body.PortalId = %d, want %d", event.Body.PortalId, portalId)
	}
}

func TestEnableActionsProvider_DifferentParameters(t *testing.T) {
	tests := []struct {
		name        string
		worldId     world.Id
		channelId   channel.Id
		mapId       _map.Id
		characterId uint32
	}{
		{"minimum values", 0, 0, 0, 0},
		{"typical values", 1, 2, 100000000, 12345},
		{"max values", 255, 255, 999999999, 4294967295},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := field.NewBuilder(tt.worldId, tt.channelId, tt.mapId).SetInstance(uuid.New()).Build()
			provider := enableActionsProvider(f, tt.characterId)
			messages, err := provider()

			if err != nil {
				t.Fatalf("enableActionsProvider() returned unexpected error: %v", err)
			}

			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}

			var event statusEvent[statusEventStatChangedBody]
			if err := json.Unmarshal(messages[0].Value, &event); err != nil {
				t.Fatalf("failed to unmarshal message value: %v", err)
			}

			if event.CharacterId != tt.characterId {
				t.Errorf("CharacterId = %d, want %d", event.CharacterId, tt.characterId)
			}

			if event.WorldId != tt.worldId {
				t.Errorf("WorldId = %d, want %d", event.WorldId, tt.worldId)
			}

			if event.Body.ChannelId != tt.channelId {
				t.Errorf("Body.ChannelId = %d, want %d", event.Body.ChannelId, tt.channelId)
			}
		})
	}
}

func TestChangeMapProvider_DifferentParameters(t *testing.T) {
	tests := []struct {
		name        string
		worldId     world.Id
		channelId   channel.Id
		mapId       _map.Id
		targetMapId _map.Id
		characterId uint32
		portalId    uint32
	}{
		{"minimum values", 0, 0, 0, 0, 0, 0},
		{"typical values", 1, 2, 100000000, 200000000, 67890, 5},
		{"max values", 255, 255, 999999999, 999999999, 4294967295, 4294967295},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := field.NewBuilder(tt.worldId, tt.channelId, tt.mapId).SetInstance(uuid.New()).Build()
			provider := ChangeMapProvider(f, tt.characterId, tt.targetMapId, tt.portalId)
			messages, err := provider()

			if err != nil {
				t.Fatalf("ChangeMapProvider() returned unexpected error: %v", err)
			}

			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}

			var event commandEvent[changeMapBody]
			if err := json.Unmarshal(messages[0].Value, &event); err != nil {
				t.Fatalf("failed to unmarshal message value: %v", err)
			}

			if event.CharacterId != tt.characterId {
				t.Errorf("CharacterId = %d, want %d", event.CharacterId, tt.characterId)
			}

			if event.WorldId != tt.worldId {
				t.Errorf("WorldId = %d, want %d", event.WorldId, tt.worldId)
			}

			if event.Body.MapId != tt.targetMapId {
				t.Errorf("Body.MapId = %d, want %d", event.Body.MapId, tt.targetMapId)
			}
		})
	}
}
