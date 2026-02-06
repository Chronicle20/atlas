package invite

import (
	invite2 "atlas-channel/kafka/message/invite"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func AcceptInviteCommandProvider(actorId uint32, worldId world.Id, inviteType string, referenceId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &invite2.Command[invite2.AcceptCommandBody]{
		WorldId:    worldId,
		InviteType: invite.Type(inviteType),
		Type:       invite.CommandTypeAccept,
		Body: invite2.AcceptCommandBody{
			ReferenceId: invite.Id(referenceId),
			TargetId:    character.Id(actorId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RejectInviteCommandProvider(actorId uint32, worldId world.Id, inviteType string, originatorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &invite2.Command[invite2.RejectCommandBody]{
		WorldId:    worldId,
		InviteType: invite.Type(inviteType),
		Type:       invite.CommandTypeReject,
		Body: invite2.RejectCommandBody{
			OriginatorId: character.Id(originatorId),
			TargetId:     character.Id(actorId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
