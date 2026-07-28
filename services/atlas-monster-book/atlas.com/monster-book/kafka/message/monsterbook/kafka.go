package monsterbook

import "github.com/google/uuid"

const (
	// Inbound (commands)
	EnvCommandTopic = "COMMAND_TOPIC_MONSTER_BOOK"

	CommandTypeCardPickedUp = "CARD_PICKED_UP"
	CommandTypeSetCover     = "SET_COVER"

	// Outbound (statuses)
	EnvEventTopicStatus = "EVENT_TOPIC_MONSTER_BOOK_STATUS"

	StatusEventTypeCardAdded    = "CARD_ADDED"
	StatusEventTypeCoverChanged = "COVER_CHANGED"
	StatusEventTypeStatsChanged = "STATS_CHANGED"
)

// Command<T> is the inbound command envelope.
type Command[B any] struct {
	TenantId    uuid.UUID `json:"tenantId"`
	CharacterId uint32    `json:"characterId"`
	EventId     uuid.UUID `json:"eventId"`
	Type        string    `json:"type"`
	Body        B         `json:"body"`
}

type CardPickedUpBody struct {
	CardId uint32 `json:"cardId"`
	Source string `json:"source"`
}

type SetCoverBody struct {
	CoverCardId uint32 `json:"coverCardId"`
}

// StatusEvent<T> is the outbound status envelope.
type StatusEvent[B any] struct {
	TenantId    uuid.UUID `json:"tenantId"`
	CharacterId uint32    `json:"characterId"`
	EventId     uuid.UUID `json:"eventId"`
	Type        string    `json:"type"`
	Body        B         `json:"body"`
}

type CardAddedBody struct {
	CardId   uint32 `json:"cardId"`
	NewLevel uint8  `json:"newLevel"`
	Full     bool   `json:"full"`
}

type CoverChangedBody struct {
	CoverCardId uint32 `json:"coverCardId"`
}

type StatsChangedBody struct {
	BookLevel        uint16 `json:"bookLevel"`
	NormalCount      uint16 `json:"normalCount"`
	SpecialCount     uint16 `json:"specialCount"`
	TotalUniqueCards uint16 `json:"totalUniqueCards"`
	ExpBonusPercent  uint16 `json:"expBonusPercent"`
}
