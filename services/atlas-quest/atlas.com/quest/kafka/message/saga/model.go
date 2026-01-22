package saga

import (
	"time"

	"github.com/google/uuid"
)

// Saga represents a saga command sent to saga-orchestrator
type Saga struct {
	TransactionId uuid.UUID `json:"transactionId"`
	SagaType      Type      `json:"sagaType"`
	InitiatedBy   string    `json:"initiatedBy"`
	Steps         []Step    `json:"steps"`
}

// Step represents a single step in a saga
type Step struct {
	Id      string `json:"id"`
	Status  Status `json:"status"`
	Action  Action `json:"action"`
	Payload any    `json:"payload"`
}

// AwardItemPayload represents payload for awarding items
type AwardItemPayload struct {
	CharacterId uint32     `json:"characterId"`
	Item        ItemDetail `json:"item"`
}

// ItemDetail represents item details for awards
type ItemDetail struct {
	TemplateId uint32    `json:"templateId"`
	Quantity   uint32    `json:"quantity"`
	Period     uint32    `json:"period,omitempty"`
	Expiration time.Time `json:"expiration,omitempty"`
}

// AwardMesosPayload represents payload for awarding mesos
type AwardMesosPayload struct {
	CharacterId uint32 `json:"characterId"`
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	ActorId     uint32 `json:"actorId"`
	ActorType   string `json:"actorType"`
	Amount      int32  `json:"amount"`
}

// AwardExperiencePayload represents payload for awarding experience
type AwardExperiencePayload struct {
	CharacterId   uint32                   `json:"characterId"`
	WorldId       byte                     `json:"worldId"`
	ChannelId     byte                     `json:"channelId"`
	Distributions []ExperienceDistribution `json:"distributions"`
}

// ExperienceDistribution represents how experience is distributed
type ExperienceDistribution struct {
	ExperienceType string `json:"experienceType"`
	Amount         uint32 `json:"amount"`
	Attr1          uint32 `json:"attr1"`
}

// AwardFamePayload represents payload for awarding fame
type AwardFamePayload struct {
	CharacterId uint32 `json:"characterId"`
	WorldId     byte   `json:"worldId"`
	ChannelId   byte   `json:"channelId"`
	ActorId     uint32 `json:"actorId"`
	ActorType   string `json:"actorType"`
	Amount      int16  `json:"amount"`
}

// CreateSkillPayload represents payload for creating/granting a skill
type CreateSkillPayload struct {
	CharacterId uint32    `json:"characterId"`
	SkillId     uint32    `json:"skillId"`
	Level       byte      `json:"level"`
	MasterLevel byte      `json:"masterLevel"`
	Expiration  time.Time `json:"expiration,omitempty"`
}

// ConsumeItemPayload represents payload for consuming/removing items
type ConsumeItemPayload struct {
	CharacterId uint32 `json:"characterId"`
	TemplateId  uint32 `json:"templateId"`
	Quantity    uint32 `json:"quantity"`
}
