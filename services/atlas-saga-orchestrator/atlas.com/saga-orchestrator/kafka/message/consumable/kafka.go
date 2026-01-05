package consumable

const (
	EnvCommandTopic = "COMMAND_TOPIC_CONSUMABLE"

	CommandApplyConsumableEffect = "APPLY_CONSUMABLE_EFFECT"
)

// Command represents a Kafka command for consumable operations
type Command[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

// ApplyConsumableEffectBody is the body for applying consumable effects without consuming from inventory
type ApplyConsumableEffectBody struct {
	ItemId uint32 `json:"itemId"`
}
