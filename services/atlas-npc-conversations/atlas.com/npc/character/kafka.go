package character

const (
	EnvCommandTopic          = "COMMAND_TOPIC_CHARACTER"
	CommandRequestChangeMeso = "REQUEST_CHANGE_MESO"
	CommandRequestChangeFame = "REQUEST_CHANGE_FAME"
	CommandChangeMap         = "CHANGE_MAP"
)

type commandEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type changeMapBody struct {
	ChannelId byte   `json:"channelId"`
	MapId     uint32 `json:"mapId"`
	PortalId  uint32 `json:"portalId"`
}

type requestChangeMesoBody struct {
	Amount int32 `json:"amount"`
}

type requestChangeFameBody struct {
	ActorId   uint32 `json:"actorId"`
	ActorType string `json:"actorType"`
	Amount    int8   `json:"amount"`
}
