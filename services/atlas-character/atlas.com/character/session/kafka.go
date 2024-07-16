package session

import (
	"atlas-character/tenant"
	"github.com/google/uuid"
)

const (
	EnvEventTopicSessionStatus      = "EVENT_TOPIC_SESSION_STATUS"
	EventSessionStatusIssuerLogin   = "LOGIN"
	EventSessionStatusIssuerChannel = "CHANNEL"
	EventSessionStatusTypeCreated   = "CREATED"
	EventSessionStatusTypeDestroyed = "DESTROYED"
)

type statusEvent struct {
	Tenant      tenant.Model `json:"tenant"`
	SessionId   uuid.UUID    `json:"sessionId"`
	AccountId   uint32       `json:"accountId"`
	CharacterId uint32       `json:"characterId"`
	WorldId     byte         `json:"worldId"`
	ChannelId   byte         `json:"channelId"`
	Issuer      string       `json:"issuer"`
	Type        string       `json:"type"`
}
