package invite

import (
	"github.com/google/uuid"
)

const (
	EnvCommandTopic         = "COMMAND_TOPIC_INVITE"
	CommandInviteTypeCreate = "CREATE"
	CommandInviteTypeAccept = "ACCEPT"
	CommandInviteTypeReject = "REJECT"

	EnvEventStatusTopic           = "EVENT_TOPIC_INVITE_STATUS"
	EventInviteStatusTypeCreated  = "CREATED"
	EventInviteStatusTypeAccepted = "ACCEPTED"
	EventInviteStatusTypeRejected = "REJECTED"

	InviteTypeBuddy        = "BUDDY"
	InviteTypeFamily       = "FAMILY"
	InviteTypeFamilySummon = "FAMILY_SUMMON"
	InviteTypeMessenger    = "MESSENGER"
	InviteTypeTrade        = "TRADE"
	InviteTypeParty        = "PARTY"
	InviteTypeGuild        = "GUILD"
	InviteTypeAlliance     = "ALLIANCE"
)

type CommandEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	InviteType    string    `json:"inviteType"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreateCommandBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
	ReferenceId  uint32 `json:"referenceId"`
}

type AcceptCommandBody struct {
	TargetId    uint32 `json:"targetId"`
	ReferenceId uint32 `json:"referenceId"`
}

type RejectCommandBody struct {
	TargetId     uint32 `json:"targetId"`
	OriginatorId uint32 `json:"originatorId"`
}

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId"`
	WorldId       byte      `json:"worldId"`
	InviteType    string    `json:"inviteType"`
	ReferenceId   uint32    `json:"referenceId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
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
