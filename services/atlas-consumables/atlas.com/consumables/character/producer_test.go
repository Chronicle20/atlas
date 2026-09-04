package character

import (
	character2 "atlas-consumables/kafka/message/character"
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
)

func TestCreditStoredExperienceCommandProvider(t *testing.T) {
	worldId := world.Id(1)
	channelId := channel.Id(2)
	characterId := uint32(1234)
	amount := uint32(3000)
	reason := "SOLOMON_ITEM"

	f := field.NewBuilder(worldId, channelId, _map.Id(0)).Build()

	provider := creditStoredExperienceCommandProvider(f, characterId, amount, reason)
	messages, err := provider()
	if err != nil {
		t.Fatalf("creditStoredExperienceCommandProvider() returned unexpected error: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	msg := messages[0]

	if !bytes.Equal(msg.Key, producer.CreateKey(int(characterId))) {
		t.Errorf("message key = %v, want %v", msg.Key, producer.CreateKey(int(characterId)))
	}

	var command character2.Command[character2.CreditStoredExperienceCommandBody]
	if err := json.Unmarshal(msg.Value, &command); err != nil {
		t.Fatalf("failed to unmarshal message value: %v", err)
	}

	if command.CharacterId != characterId {
		t.Errorf("CharacterId = %d, want %d", command.CharacterId, characterId)
	}

	if command.WorldId != worldId {
		t.Errorf("WorldId = %d, want %d", command.WorldId, worldId)
	}

	if command.Type != character2.CommandCreditStoredExperience {
		t.Errorf("Type = %s, want %s", command.Type, character2.CommandCreditStoredExperience)
	}

	if command.Body.ChannelId != channelId {
		t.Errorf("Body.ChannelId = %d, want %d", command.Body.ChannelId, channelId)
	}

	if command.Body.Amount != amount {
		t.Errorf("Body.Amount = %d, want %d", command.Body.Amount, amount)
	}

	if command.Body.Reason != reason {
		t.Errorf("Body.Reason = %s, want %s", command.Body.Reason, reason)
	}
}
