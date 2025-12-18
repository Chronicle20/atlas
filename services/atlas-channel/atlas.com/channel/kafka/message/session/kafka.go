package session

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvEventTopicSessionStatus      = "EVENT_TOPIC_SESSION_STATUS"
	EventSessionStatusIssuerLogin   = "LOGIN"
	EventSessionStatusIssuerChannel = "CHANNEL"
	EventSessionStatusTypeCreated   = "CREATED"
	EventSessionStatusTypeDestroyed = "DESTROYED"
)

type StatusEvent struct {
	SessionId   uuid.UUID  `json:"sessionId"`
	AccountId   uint32     `json:"accountId"`
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	Issuer      string     `json:"issuer"`
	Type        string     `json:"type"`
}
