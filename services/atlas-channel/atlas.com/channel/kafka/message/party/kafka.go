package party

import (
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandTopic           = "COMMAND_TOPIC_PARTY"
	CommandPartyCreate        = "CREATE"
	CommandPartyLeave         = "LEAVE"
	CommandPartyChangeLeader  = "CHANGE_LEADER"
	CommandPartyRequestInvite = "REQUEST_INVITE"
)

type Command[E any] struct {
	ActorId uint32 `json:"actorId"`
	Type    string `json:"type"`
	Body    E      `json:"body"`
}

type CreateCommandBody struct {
}

type LeaveCommandBody struct {
	PartyId uint32 `json:"partyId"`
	Force   bool   `json:"force"`
}

type ChangeLeaderBody struct {
	PartyId  uint32 `json:"partyId"`
	LeaderId uint32 `json:"leaderId"`
}

type RequestInviteBody struct {
	CharacterId uint32 `json:"characterId"`
}

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

type CreatedEventBody struct {
}

type JoinedEventBody struct {
}

type LeftEventBody struct {
}

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

type ErrorEventBody struct {
	Type          string `json:"type"`
	CharacterName string `json:"characterName"`
}
