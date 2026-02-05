package invite

import (
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-constants/world"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_INVITE"
)

type commandEvent[E any] struct {
	WorldId    world.Id           `json:"worldId"`
	InviteType invite.Type        `json:"inviteType"`
	Type       invite.CommandType `json:"type"`
	Body       E                  `json:"body"`
}

type createCommandBody struct {
	OriginatorId character.Id `json:"originatorId"`
	TargetId     character.Id `json:"targetId"`
	ReferenceId  invite.Id    `json:"referenceId"`
}
