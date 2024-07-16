package character

import (
	"atlas-character/tenant"
)

const (
	EnvEventTopicCharacterStatus    = "EVENT_TOPIC_CHARACTER_STATUS"
	EventCharacterStatusTypeCreated = "CREATED"
	EventCharacterStatusTypeLogin   = "LOGIN"
	EventCharacterStatusTypeLogout  = "LOGOUT"
)

type statusEvent struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Name        string       `json:"name"`
	WorldId     byte         `json:"worldId"`
	ChannelId   byte         `json:"channelId"`
	MapId       uint32       `json:"mapId"`
	Type        string       `json:"type"`
}
