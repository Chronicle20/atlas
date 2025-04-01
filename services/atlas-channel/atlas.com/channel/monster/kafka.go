package monster

const (
	EnvCommandTopic   = "COMMAND_TOPIC_MONSTER"
	CommandTypeDamage = "DAMAGE"
)

type command[E any] struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MonsterId uint32 `json:"monsterId"`
	Type      string `json:"type"`
	Body      E      `json:"body"`
}

type damageCommandBody struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
}
