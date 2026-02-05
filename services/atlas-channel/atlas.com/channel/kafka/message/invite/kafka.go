package invite

import (
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_INVITE"

	EnvEventStatusTopic = "EVENT_TOPIC_INVITE_STATUS"
)

type Command[E any] struct {
	WorldId    world.Id           `json:"worldId"`
	InviteType invite.Type        `json:"inviteType"`
	Type       invite.CommandType `json:"type"`
	Body       E                  `json:"body"`
}

type AcceptCommandBody struct {
	TargetId    character.Id `json:"targetId"`
	ReferenceId invite.Id    `json:"referenceId"`
}

type RejectCommandBody struct {
	TargetId     character.Id `json:"targetId"`
	OriginatorId character.Id `json:"originatorId"`
}

type StatusEvent[E any] struct {
	WorldId     world.Id          `json:"worldId"`
	InviteType  invite.Type       `json:"inviteType"`
	ReferenceId invite.Id         `json:"referenceId"`
	Type        invite.StatusType `json:"type"`
	Body        E                 `json:"body"`
}

type CreatedEventBody struct {
	OriginatorId character.Id `json:"originatorId"`
	TargetId     character.Id `json:"targetId"`
}

type AcceptedEventBody struct {
	OriginatorId character.Id `json:"originatorId"`
	TargetId     character.Id `json:"targetId"`
}

type RejectedEventBody struct {
	OriginatorId character.Id `json:"originatorId"`
	TargetId     character.Id `json:"targetId"`
}
