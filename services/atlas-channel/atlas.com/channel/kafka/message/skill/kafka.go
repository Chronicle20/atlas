package skill

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_SKILL"
)

const (
	CommandTypeSetCooldown    = "SET_COOLDOWN"
	CommandTypeResetCooldowns = "RESET_COOLDOWNS"
)

// Command mirrors atlas-skills' command envelope field-for-field.
// TransactionId/WorldId were added with RESET_COOLDOWNS (task-155);
// SET_COOLDOWN keeps emitting zero values for both — the same values the
// skills-side decoder produced for the old, field-less JSON.
type Command[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type SetCooldownBody struct {
	SkillId  uint32 `json:"skillId"`
	Cooldown uint32 `json:"cooldown"`
}

// ResetCooldownsBody clears every active cooldown for the character except
// the listed skill ids. SourceSkillId identifies the triggering skill
// (5121010 for Time Leap) and is observability-only.
type ResetCooldownsBody struct {
	ExceptSkillIds []uint32 `json:"exceptSkillIds"`
	SourceSkillId  uint32   `json:"sourceSkillId"`
}

const (
	EnvStatusEventTopic topic.Token = "EVENT_TOPIC_SKILL_STATUS"
)

const (
	StatusEventTypeCreated         = "CREATED"
	StatusEventTypeUpdated         = "UPDATED"
	StatusEventTypeDeleted         = "DELETED"
	StatusEventTypeCooldownApplied = "COOLDOWN_APPLIED"
	StatusEventTypeCooldownExpired = "COOLDOWN_EXPIRED"
)

type StatusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint32 `json:"skillId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
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

type StatusEventCooldownAppliedBody struct {
	CooldownExpiresAt time.Time `json:"cooldownExpiresAt"`
}

type StatusEventCooldownExpiredBody struct{}

// StatusEventDeletedBody is the empty body emitted alongside StatusEventTypeDeleted.
type StatusEventDeletedBody struct{}
