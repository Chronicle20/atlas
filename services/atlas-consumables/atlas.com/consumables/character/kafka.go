package character

const (
	EnvCommandTopic = "COMMAND_TOPIC_CHARACTER"
	CommandChangeHP = "CHANGE_HP"
	CommandChangeMP = "CHANGE_MP"
)

type command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type changeHPCommandBody struct {
	Amount int16 `json:"amount"`
}

type changeMPCommandBody struct {
	Amount int16 `json:"amount"`
}
