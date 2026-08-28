package consumable

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_CONSUMABLE"

	CommandApplyConsumableEffect  = "APPLY_CONSUMABLE_EFFECT"
	CommandCancelConsumableEffect = "CANCEL_CONSUMABLE_EFFECT"
)

// Command mirrors atlas-consumables' envelope field-for-field, including the
// MapId and Instance this service always leaves zero. Keeping the shapes
// identical means a future field addition on the consumer side is a visible
// diff here rather than a silent decode of the wrong layout.
//
// This service emits only the two effect commands, so only their bodies are
// declared. Every service on this topic keeps its own copy of the contract
// (atlas-saga-orchestrator does the same); importing another service's Go
// package would break the service boundary.
type Command[E any] struct {
	TransactionId uuid.UUID    `json:"transactionId"`
	WorldId       world.Id     `json:"worldId"`
	ChannelId     channel.Id   `json:"channelId"`
	MapId         _map.Id      `json:"mapId"`
	Instance      uuid.UUID    `json:"instance"`
	CharacterId   character.Id `json:"characterId"`
	Type          string       `json:"type"`
	Body          E            `json:"body"`
}

// ApplyConsumableEffectBody applies a consumable's effect without consuming
// anything from inventory.
type ApplyConsumableEffectBody struct {
	ItemId item.Id `json:"itemId"`
}

// CancelConsumableEffectBody cancels a previously applied consumable effect.
type CancelConsumableEffectBody struct {
	ItemId item.Id `json:"itemId"`
}
