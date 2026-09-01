// Package party mirrors the wire contract atlas-parties actually decodes.
// atlas-parties carries two slightly different local copies of this
// contract (party/kafka.go for producing, kafka/consumer/party/kafka.go for
// consuming); the consumer-side shape is authoritative for what a LEAVE
// command must contain, since that is what atlas-parties' handleLeave reads
// (services/atlas-parties/atlas.com/parties/kafka/consumer/party/kafka.go).
package party

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_PARTY"
)

const (
	CommandTypeLeave = "LEAVE"
)

const (
	EnvStatusEventTopic topic.Token = "EVENT_TOPIC_PARTY_STATUS"
)

const (
	StatusEventTypeLeft    = "LEFT"
	StatusEventTypeDisband = "DISBAND"
)

type Command[E any] struct {
	ActorId       uint32    `json:"actorId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}

// LeaveBody mirrors kafka/consumer/party/kafka.go's leaveCommandBody in
// atlas-parties, NOT party/kafka.go's (that copy lacks CharacterId; the
// consumer reads CharacterId to know which member to remove). CharacterId
// equals ActorId for a self-initiated (non-forced) leave, which is the only
// mode the world-transfer saga uses.
type LeaveBody struct {
	PartyId     uint32 `json:"partyId"`
	Force       bool   `json:"force"`
	CharacterId uint32 `json:"characterId"`
}

type StatusEvent[E any] struct {
	ActorId       uint32    `json:"actorId"`
	WorldId       world.Id  `json:"worldId"`
	PartyId       uint32    `json:"partyId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
}

type StatusEventLeftBody struct{}

// StatusEventDisbandBody is emitted instead of LEFT when the leaving
// character is the party leader (atlas-parties disbands rather than
// transferring leadership). The world-transfer saga must accept this as an
// alternate completion of leave_party_for_transfer, or a leader's transfer
// hangs to timeout even though the party was in fact vacated.
type StatusEventDisbandBody struct {
	Members []uint32 `json:"members"`
}
