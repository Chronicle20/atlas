package invite

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic         = "COMMAND_TOPIC_INVITE"
	CommandInviteTypeCreate = "CREATE"

	EnvEventStatusTopic           = "EVENT_TOPIC_INVITE_STATUS"
	EventInviteStatusTypeAccepted = "ACCEPTED"

	InviteTypeMessenger = "MESSENGER"
)

type CommandEvent[E any] struct {
	TransactionID uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	InviteType    string    `json:"inviteType"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreateCommandBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
	ReferenceId  uint32 `json:"referenceId"`
}

type StatusEvent[E any] struct {
	TransactionID uuid.UUID `json:"transactionId"`
	WorldId       world.Id  `json:"worldId"`
	InviteType    string    `json:"inviteType"`
	ReferenceId   uint32    `json:"referenceId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type AcceptedEventBody struct {
	OriginatorId uint32 `json:"originatorId"`
	TargetId     uint32 `json:"targetId"`
}
