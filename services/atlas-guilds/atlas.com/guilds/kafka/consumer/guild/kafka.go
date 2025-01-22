package guild

const (
	EnvCommandTopic              = "COMMAND_TOPIC_GUILD"
	CommandTypeRequestCreate     = "REQUEST_CREATE"
	CommandTypeCreationAgreement = "CREATION_AGREEMENT"
	CommandTypeChangeEmblem      = "CHANGE_EMBLEM"
)

type command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type requestCreateBody struct {
	WorldId   byte   `json:"worldId"`
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	Name      string `json:"name"`
}

type creationAgreementBody struct {
	Agreed bool `json:"agreed"`
}

type changeEmblemBody struct {
	GuildId             uint32 `json:"guildId"`
	Logo                uint16 `json:"logo"`
	LogoColor           byte   `json:"logoColor"`
	LogoBackground      uint16 `json:"logoBackground"`
	LogoBackgroundColor byte   `json:"logoBackgroundColor"`
}
