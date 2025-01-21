package guild

const (
	EnvCommandTopic        = "COMMAND_TOPIC_GUILD"
	CommandTypeRequestName = "REQUEST_NAME"

	EnvStatusEventTopic             = "EVENT_TOPIC_GUILD_STATUS"
	StatusEventTypeRequestAgreement = "REQUEST_AGREEMENT"
	StatusEventTypeError            = "ERROR"
)

type command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type requestNameBody struct {
	WorldId   byte `json:"worldId"`
	ChannelId byte `json:"channelId"`
}

type statusEvent[E any] struct {
	WorldId byte   `json:"worldId"`
	GuildId uint32 `json:"guildId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type statusEventRequestAgreementBody struct {
	ActorId      uint32 `json:"actorId"`
	ProposedName string `json:"proposedName"`
}

type statusEventErrorBody struct {
	ActorId uint32 `json:"actorId"`
	Error   string `json:"error"`
}
