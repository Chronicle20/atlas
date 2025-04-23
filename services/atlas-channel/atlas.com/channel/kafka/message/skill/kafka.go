package skill

import "time"

const (
	EnvCommandTopic        = "COMMAND_TOPIC_SKILL"
	CommandTypeSetCooldown = "SET_COOLDOWN"
)

type Command[E any] struct {
	CharacterId uint32 `json:"characterId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type SetCooldownBody struct {
	SkillId  uint32 `json:"skillId"`
	Cooldown uint32 `json:"cooldown"`
}

const (
	EnvStatusEventTopic            = "EVENT_TOPIC_SKILL_STATUS"
	StatusEventTypeCreated         = "CREATED"
	StatusEventTypeUpdated         = "UPDATED"
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

type StatusEventCooldownExpiredBody struct {
}
