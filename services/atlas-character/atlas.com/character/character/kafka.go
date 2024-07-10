package character

import (
	"atlas-character/tenant"
)

const (
	EnvEventTopicCharacterCreated = "EVENT_TOPIC_CHARACTER_CREATED"
)

type createdEvent struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	WorldId     byte         `json:"worldId"`
	Name        string       `json:"name"`
}
