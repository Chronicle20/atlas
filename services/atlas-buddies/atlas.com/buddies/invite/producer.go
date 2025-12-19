package invite

import (
	invite2 "atlas-buddies/kafka/message/invite"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(actorId uint32, worldId byte, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(targetId))
	value := &invite2.Command[invite2.CreateCommandBody]{
		WorldId:    worldId,
		InviteType: invite2.InviteTypeBuddy,
		Type:       invite2.CommandInviteTypeCreate,
		Body: invite2.CreateCommandBody{
			OriginatorId: actorId,
			TargetId:     targetId,
			ReferenceId:  actorId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func rejectInviteCommandProvider(actorId uint32, worldId byte, originatorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &invite2.Command[invite2.RejectCommandBody]{
		WorldId:    worldId,
		InviteType: invite2.InviteTypeBuddy,
		Type:       invite2.CommandInviteTypeReject,
		Body: invite2.RejectCommandBody{
			OriginatorId: originatorId,
			TargetId:     actorId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
