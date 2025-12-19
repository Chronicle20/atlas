package buff

const (
	EnvCommandTopic   = "COMMAND_TOPIC_CHARACTER_BUFF"
	CommandTypeApply  = "APPLY"
	CommandTypeCancel = "CANCEL"
)

type Command[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type ApplyCommandBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Duration int32        `json:"duration"`
	Changes  []StatChange `json:"changes"`
}

type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type CancelCommandBody struct {
	SourceId int32 `json:"sourceId"`
}
