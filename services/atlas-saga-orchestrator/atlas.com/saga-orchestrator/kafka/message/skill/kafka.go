package skill

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic          = "COMMAND_TOPIC_SKILL"
	CommandTypeRequestCreate = "REQUEST_CREATE"
	CommandTypeRequestUpdate = "REQUEST_UPDATE"
	CommandTypeRequestDelete = "REQUEST_DELETE"
	CommandTypeTransferSp    = "TRANSFER_SP"
)

// TransferSpBody moves one skill point FromSkillId -> ToSkillId (SP Reset
// item 505000<ItemTier>). JobId and TargetMaxLevel are supplied by the
// trusted server-side caller because atlas-skills stores neither job nor
// game data; everything state-derived is re-validated there.
type TransferSpBody struct {
	JobId          job.Id `json:"jobId"`
	FromSkillId    uint32 `json:"fromSkillId"`
	ToSkillId      uint32 `json:"toSkillId"`
	ItemTier       byte   `json:"itemTier"`
	TargetMaxLevel byte   `json:"targetMaxLevel"`
}

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
	EnvStatusEventTopic          = "EVENT_TOPIC_SKILL_STATUS"
	StatusEventTypeCreated       = "CREATED"
	StatusEventTypeUpdated       = "UPDATED"
	StatusEventTypeDeleted       = "DELETED"
	StatusEventTypeSpTransferred = "SP_TRANSFERRED"
	StatusEventTypeError         = "ERROR"

	StatusEventErrorTypeSkillAtZero   = "SKILL_AT_ZERO"
	StatusEventErrorTypeSkillAtCap    = "SKILL_AT_CAP"
	StatusEventErrorTypeWrongTier     = "WRONG_TIER"
	StatusEventErrorTypeInvalidTarget = "INVALID_TARGET"
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

// StatusEventSpTransferredBody signals a completed SP transfer; the envelope
// SkillId carries the target skill. This is the saga-completion event.
type StatusEventSpTransferredBody struct {
	FromSkillId uint32 `json:"fromSkillId"`
	FromLevel   byte   `json:"fromLevel"`
	ToLevel     byte   `json:"toLevel"`
}

// StatusEventErrorBody reports a rejected TRANSFER_SP; Error is one of the
// StatusEventErrorType* constants, Detail names the offending skill id.
type StatusEventErrorBody struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}
