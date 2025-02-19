package skill

import "time"

const (
	EnvStatusEventTopic            = "EVENT_TOPIC_SKILL_STATUS"
	StatusEventTypeCreated         = "CREATED"
	StatusEventTypeUpdated         = "UPDATED"
	StatusEventTypeCooldownApplied = "COOLDOWN_APPLIED"
	StatusEventTypeCooldownExpired = "COOLDOWN_EXPIRED"
)

type statusEvent[E any] struct {
	CharacterId uint32 `json:"characterId"`
	SkillId     uint32 `json:"skillId"`
	Type        string `json:"type"`
	Body        E      `json:"body"`
}

type statusEventCreatedBody struct {
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration"`
}

type statusEventUpdatedBody struct {
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration"`
}

type statusEventCooldownAppliedBody struct {
	CooldownExpiresAt time.Time `json:"cooldownExpiresAt"`
}

type statusEventCooldownExpiredBody struct {
}
