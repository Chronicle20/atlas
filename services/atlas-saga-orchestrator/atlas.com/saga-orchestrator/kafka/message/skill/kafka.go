package skill

import (
	"time"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic          = "COMMAND_TOPIC_SKILL"
	CommandTypeRequestCreate = "REQUEST_CREATE"
	CommandTypeRequestUpdate = "REQUEST_UPDATE"
	CommandTypeRequestDelete = "REQUEST_DELETE"
)

type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type RequestCreateBody struct {
	SkillId     uint32    `json:"skillId"`
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration"`
}

type RequestUpdateBody struct {
	SkillId     uint32    `json:"skillId"`
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration"`
}

// RequestDeleteBody is the saga-correlated REQUEST_DELETE command body used
// by the character-creation reverse-walk compensator (plan Phase 5 / 6).
type RequestDeleteBody struct {
	SkillId uint32 `json:"skillId"`
}

const (
	EnvStatusEventTopic    = "EVENT_TOPIC_SKILL_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeUpdated = "UPDATED"
	StatusEventTypeDeleted = "DELETED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	SkillId       uint32    `json:"skillId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type StatusEventCreatedBody struct {
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration"`
}

type StatusEventUpdatedBody struct {
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration"`
}

// StatusEventDeletedBody is the empty body emitted alongside StatusEventTypeDeleted.
type StatusEventDeletedBody struct{}
