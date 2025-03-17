package character

const (
	EnvCommandTopic   = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply  = "APPLY"
	CommandTypeCancel = "CANCEL"
)

type command[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type applyCommandBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Duration int32        `json:"duration"`
	Changes  []statChange `json:"changes"`
}

type statChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type cancelCommandBody struct {
	SourceId int32 `json:"sourceId"`
}
