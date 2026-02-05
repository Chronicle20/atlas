package invite

import (
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvEventStatusTopic = "EVENT_TOPIC_INVITE_STATUS"
)

type StatusEvent[E any] struct {
	WorldId     world.Id          `json:"worldId"`
	InviteType  invite.Type       `json:"inviteType"`
	ReferenceId invite.Id         `json:"referenceId"`
	Type        invite.StatusType `json:"type"`
	Body        E                 `json:"body"`
}

type AcceptedEventBody struct {
	OriginatorId character.Id `json:"originatorId"`
	TargetId     character.Id `json:"targetId"`
}
