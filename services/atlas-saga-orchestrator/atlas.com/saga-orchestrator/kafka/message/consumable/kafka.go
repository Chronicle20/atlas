package consumable

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_CONSUMABLE"

	CommandApplyConsumableEffect = "APPLY_CONSUMABLE_EFFECT"

	// Consumable status event constants
	EnvEventTopicStatus             = "EVENT_TOPIC_CONSUMABLE_STATUS"
	StatusEventTypeEffectApplied    = "EFFECT_APPLIED"
)

// Command represents a Kafka command for consumable operations
type Command[E any] struct {
	TransactionId uuid.UUID    `json:"transactionId"`
	WorldId       world.Id     `json:"worldId"`
	ChannelId     channel.Id   `json:"channelId"`
	CharacterId   character.Id `json:"characterId"`
	Type          string       `json:"type"`
	Body          E            `json:"body"`
}

// ApplyConsumableEffectBody is the body for applying consumable effects without consuming from inventory
type ApplyConsumableEffectBody struct {
	ItemId item.Id `json:"itemId"`
}

// StatusEvent represents a consumable status event
type StatusEvent[E any] struct {
	CharacterId character.Id `json:"characterId"`
	Type        string       `json:"type"`
	Body        E            `json:"body"`
}

// EffectAppliedStatusEventBody represents the body of an effect applied event
type EffectAppliedStatusEventBody struct {
	ItemId        item.Id   `json:"itemId"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}
