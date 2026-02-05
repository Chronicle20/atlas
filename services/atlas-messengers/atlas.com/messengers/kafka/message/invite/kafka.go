package invite

import (
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_INVITE"

	EnvEventStatusTopic = "EVENT_TOPIC_INVITE_STATUS"
)

type Command[E any] struct {
	TransactionId uuid.UUID          `json:"transactionId"`
	WorldId       world.Id           `json:"worldId"`
	InviteType    invite.Type        `json:"inviteType"`
	Type          invite.CommandType `json:"type"`
	Body          E                  `json:"body"`
}

type CreateCommandBody struct {
	OriginatorId character.Id `json:"originatorId"`
	TargetId     character.Id `json:"targetId"`
	ReferenceId  invite.Id    `json:"referenceId"`
}

type StatusEvent[E any] struct {
	TransactionId uuid.UUID         `json:"transactionId"`
	WorldId       world.Id          `json:"worldId"`
	InviteType    invite.Type       `json:"inviteType"`
	ReferenceId   invite.Id         `json:"referenceId"`
	Type          invite.StatusType `json:"type"`
	Body          E                 `json:"body"`
}

type AcceptedEventBody struct {
	OriginatorId character.Id `json:"originatorId"`
	TargetId     character.Id `json:"targetId"`
}
