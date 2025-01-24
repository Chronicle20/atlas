package character

const (
	EnvCommandTopic          = "COMMAND_TOPIC_CHARACTER"
	CommandRequestChangeMeso = "REQUEST_CHANGE_MESO"
)

type commandEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type requestChangeMesoBody struct {
	Amount int32 `json:"amount"`
}
