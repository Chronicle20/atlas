package instance

import (
	"atlas-transports/kafka/message/consumable"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

// TestConsumableEffectProviders_WireShape tables the apply/cancel wire-shape
// cases: both call the same decode-and-assert shape against a provider of
// identical signature, differing only in which provider constructor is used
// and the expected command type. Every assertion from both original
// standalone tests is preserved (including TransactionId==uuid.Nil, which
// the source confirms applies identically to both APPLY and CANCEL).
func TestConsumableEffectProviders_WireShape(t *testing.T) {
	tests := []struct {
		name        string
		newProvider func(world.Id, channel.Id, uint32, item.Id) model.Provider[[]kafka.Message]
		wantType    string
	}{
		{
			name:        "ApplyConsumableEffectProvider_WireShape",
			newProvider: applyConsumableEffectProvider,
			wantType:    consumable.CommandApplyConsumableEffect,
		},
		{
			name:        "CancelConsumableEffectProvider_WireShape",
			newProvider: cancelConsumableEffectProvider,
			wantType:    consumable.CommandCancelConsumableEffect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, err := tt.newProvider(world.Id(0), channel.Id(1), 42, item.Id(2210016))()
			assert.NoError(t, err)
			assert.Len(t, ms, 1)

			var got decodedConsumable
			assert.NoError(t, json.Unmarshal(ms[0].Value, &got))

			assert.Equal(t, tt.wantType, got.Type)
			assert.Equal(t, world.Id(0), got.WorldId)
			assert.Equal(t, channel.Id(1), got.ChannelId)
			assert.Equal(t, uint32(42), got.CharacterId)
			assert.Equal(t, item.Id(2210016), got.Body.ItemId)
			// uuid.Nil marks a non-saga effect application: atlas-saga-orchestrator
			// skips saga completion for it rather than logging an orphan transaction.
			assert.Equal(t, uuid.Nil, got.TransactionId)
		})
	}
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
