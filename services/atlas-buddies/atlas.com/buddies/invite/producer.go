package invite

import (
	invite2 "atlas-buddies/kafka/message/invite"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(actorId character.Id, worldId world.Id, targetId character.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(targetId))
	value := &invite2.Command[invite2.CreateCommandBody]{
		WorldId:    worldId,
		InviteType: invite.TypeBuddy,
		Type:       invite.CommandTypeCreate,
		Body: invite2.CreateCommandBody{
			OriginatorId: actorId,
			TargetId:     targetId,
			ReferenceId:  invite.Id(actorId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func rejectInviteCommandProvider(actorId character.Id, worldId world.Id, originatorId character.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &invite2.Command[invite2.RejectCommandBody]{
		WorldId:    worldId,
		InviteType: invite.TypeBuddy,
		Type:       invite.CommandTypeReject,
		Body: invite2.RejectCommandBody{
			OriginatorId: originatorId,
			TargetId:     actorId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
