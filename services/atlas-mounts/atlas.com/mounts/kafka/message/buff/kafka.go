package buff

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// These structs mirror the character buff-status event shapes emitted by
// atlas-buffs (services/atlas-buffs/.../kafka/message/character/kafka.go). The
// field names and json tags must stay byte-compatible with the producer.
const (
	EnvEventStatusTopic        = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"
	EventStatusTypeBuffApplied = "APPLIED"
	EventStatusTypeBuffExpired = "EXPIRED"
)

type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type AppliedStatusEventBody struct {
	FromId    uint32       `json:"fromId"`
	SourceId  int32        `json:"sourceId"`
	Level     byte         `json:"level"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}

type ExpiredStatusEventBody struct {
	SourceId  int32        `json:"sourceId"`
	Level     byte         `json:"level"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}

type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}
