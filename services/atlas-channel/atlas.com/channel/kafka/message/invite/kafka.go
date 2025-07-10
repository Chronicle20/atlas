package invite

import (
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	InviteTypeBuddy        = "BUDDY"
	InviteTypeFamily       = "FAMILY"
	InviteTypeFamilySummon = "FAMILY_SUMMON"
	InviteTypeMessenger    = "MESSENGER"
	InviteTypeTrade        = "TRADE"
	InviteTypeParty        = "PARTY"
	InviteTypeGuild        = "GUILD"
	InviteTypeAlliance     = "ALLIANCE"
)

const (
	EnvCommandTopic         = "COMMAND_TOPIC_INVITE"
	CommandInviteTypeAccept = "ACCEPT"
	CommandInviteTypeReject = "REJECT"
)

type Command[E any] struct {
	WorldId    world.Id `json:"worldId"`
	InviteType string   `json:"inviteType"`
	Type       string   `json:"type"`
	Body       E        `json:"body"`
}

type AcceptCommandBody struct {
	TargetId    uint32 `json:"targetId"`
	ReferenceId uint32 `json:"referenceId"`
}

type RejectCommandBody struct {
	TargetId     uint32 `json:"targetId"`
	OriginatorId uint32 `json:"originatorId"`
}

const (
	EnvEventStatusTopic           = "EVENT_TOPIC_INVITE_STATUS"
	EventInviteStatusTypeCreated  = "CREATED"
	EventInviteStatusTypeAccepted = "ACCEPTED"
	EventInviteStatusTypeRejected = "REJECTED"
)

type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	InviteType  string   `json:"inviteType"`
	ReferenceId uint32   `json:"referenceId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type CreatedEventBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
}

type AcceptedEventBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
}

type RejectedEventBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
}
