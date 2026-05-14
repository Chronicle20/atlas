package monsterbook

import "github.com/google/uuid"

const (
	EnvCommandTopic         = "COMMAND_TOPIC_MONSTER_BOOK"
	CommandTypeCardPickedUp = "CARD_PICKED_UP"
)

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
