// Package session carries the EVENT_TOPIC_SESSION_STATUS envelope this service
// consumes to tear a character out of its trade room when their session dies.
// Mirrors the PRODUCER, services/atlas-channel/atlas.com/channel/kafka/message/
// session/kafka.go — struct name, field names and json tags must match it
// exactly.
//
// The envelope carries no transactionId, so the consumer mints one for the
// teardown it drives.
package session

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvEventTopicSessionStatus topic.Token = "EVENT_TOPIC_SESSION_STATUS"
)

const (
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
