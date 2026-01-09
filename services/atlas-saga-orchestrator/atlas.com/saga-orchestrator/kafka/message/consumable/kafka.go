package consumable

import "github.com/google/uuid"

const (
	EnvCommandTopic = "COMMAND_TOPIC_CONSUMABLE"

	CommandApplyConsumableEffect = "APPLY_CONSUMABLE_EFFECT"

	// Consumable status event constants
	EnvEventTopicStatus             = "EVENT_TOPIC_CONSUMABLE_STATUS"
	StatusEventTypeEffectApplied    = "EFFECT_APPLIED"
)

// Command represents a Kafka command for consumable operations
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	ChannelId     byte      `json:"channelId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

// ApplyConsumableEffectBody is the body for applying consumable effects without consuming from inventory
type ApplyConsumableEffectBody struct {
	ItemId uint32 `json:"itemId"`
}

// StatusEvent represents a consumable status event
type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// EffectAppliedStatusEventBody represents the body of an effect applied event
type EffectAppliedStatusEventBody struct {
	ItemId        uint32    `json:"itemId"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}
