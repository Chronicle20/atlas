package buff

const (
	EnvEventStatusTopic        = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"
	EventStatusTypeBuffApplied = "APPLIED"
	EventStatusTypeBuffExpired = "EXPIRED"
)

type statusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type appliedStatusEventBody struct {
	SourceId uint32       `json:"sourceId"`
	Duration int32        `json:"duration"`
	Changes  []statChange `json:"changes"`
}

type statChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type expiredStatusEventBody struct {
	SourceId uint32 `json:"sourceId"`
}
