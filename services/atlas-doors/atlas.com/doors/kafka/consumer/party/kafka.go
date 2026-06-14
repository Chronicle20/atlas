package party

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// These structs mirror the party status event shapes emitted by atlas-parties.
// The field names and json tags must stay byte-compatible with the producer.
// Source of truth: services/atlas-parties/atlas.com/parties/party/kafka.go
const (
	EnvEventStatusTopic              = "EVENT_TOPIC_PARTY_STATUS"
	EventPartyStatusTypeCreated      = "CREATED"
	EventPartyStatusTypeJoined       = "JOINED"
	EventPartyStatusTypeLeft         = "LEFT"
	EventPartyStatusTypeExpel        = "EXPEL"
	EventPartyStatusTypeDisband      = "DISBAND"
	EventPartyStatusTypeChangeLeader = "CHANGE_LEADER"
	EventPartyStatusTypeError        = "ERROR"
)

type StatusEvent[E any] struct {
	ActorId uint32   `json:"actorId"`
	WorldId world.Id `json:"worldId"`
	PartyId uint32   `json:"partyId"`
	Type    string   `json:"type"`
	Body    E        `json:"body"`
}

type JoinedEventBody struct{}

type LeftEventBody struct{}

type ExpelEventBody struct {
	CharacterId uint32 `json:"characterId"`
}

type DisbandEventBody struct {
	Members []uint32 `json:"members"`
}

type ChangeLeaderEventBody struct {
	CharacterId  uint32 `json:"characterId"`
	Disconnected bool   `json:"disconnected"`
}
