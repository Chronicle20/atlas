package instance

import (
	"atlas-transports/kafka/message/consumable"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// decodedConsumable is the on-the-wire shape of both commands: APPLY and
// CANCEL share one body ({itemId}), so one decoder covers both.
type decodedConsumable struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          struct {
		ItemId item.Id `json:"itemId"`
	} `json:"body"`
}

func TestApplyConsumableEffectProvider_WireShape(t *testing.T) {
	ms, err := applyConsumableEffectProvider(world.Id(0), channel.Id(1), 42, item.Id(2210016))()
	assert.NoError(t, err)
	assert.Len(t, ms, 1)

	var got decodedConsumable
	assert.NoError(t, json.Unmarshal(ms[0].Value, &got))

	assert.Equal(t, consumable.CommandApplyConsumableEffect, got.Type)
	assert.Equal(t, world.Id(0), got.WorldId)
	assert.Equal(t, channel.Id(1), got.ChannelId)
	assert.Equal(t, uint32(42), got.CharacterId)
	assert.Equal(t, item.Id(2210016), got.Body.ItemId)
	// uuid.Nil marks a non-saga effect application: atlas-saga-orchestrator
	// skips saga completion for it rather than logging an orphan transaction.
	assert.Equal(t, uuid.Nil, got.TransactionId)
}

func TestCancelConsumableEffectProvider_WireShape(t *testing.T) {
	ms, err := cancelConsumableEffectProvider(world.Id(0), channel.Id(1), 42, item.Id(2210016))()
	assert.NoError(t, err)
	assert.Len(t, ms, 1)

	var got decodedConsumable
	assert.NoError(t, json.Unmarshal(ms[0].Value, &got))

	assert.Equal(t, consumable.CommandCancelConsumableEffect, got.Type)
	assert.Equal(t, uint32(42), got.CharacterId)
	assert.Equal(t, item.Id(2210016), got.Body.ItemId)
}

// Both commands are keyed by characterId so they land on one partition and
// atlas-consumables (serial by default) can never reorder an APPLY past a
// later CANCEL for the same character.
func TestConsumableProviders_ShareOneKeyPerCharacter(t *testing.T) {
	apply, err := applyConsumableEffectProvider(world.Id(0), channel.Id(1), 42, item.Id(2210016))()
	assert.NoError(t, err)
	cancel, err := cancelConsumableEffectProvider(world.Id(0), channel.Id(1), 42, item.Id(2210016))()
	assert.NoError(t, err)

	assert.Equal(t, apply[0].Key, cancel[0].Key)
}
