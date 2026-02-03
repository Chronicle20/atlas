package buff

import "time"

const (
	EnvEventStatusTopic        = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"
	EventStatusTypeBuffApplied = "APPLIED"
	EventStatusTypeBuffExpired = "EXPIRED"
)

type StatusEvent[E any] struct {
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

type AppliedStatusEventBody struct {
	FromId    uint32       `json:"fromId"`
	SourceId  int32        `json:"sourceId"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}

type ExpiredStatusEventBody struct {
	SourceId  int32        `json:"sourceId"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}
