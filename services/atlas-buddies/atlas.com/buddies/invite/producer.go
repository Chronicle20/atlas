package invite

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(actorId uint32, worldId byte, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(targetId))
	value := &commandEvent[createCommandBody]{
		WorldId:    worldId,
		InviteType: InviteTypeBuddy,
		Type:       CommandInviteTypeCreate,
		Body: createCommandBody{
			OriginatorId: actorId,
			TargetId:     targetId,
			ReferenceId:  actorId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func rejectInviteCommandProvider(actorId uint32, worldId byte, originatorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &commandEvent[rejectCommandBody]{
		WorldId:    worldId,
		InviteType: InviteTypeBuddy,
		Type:       CommandInviteTypeReject,
		Body: rejectCommandBody{
			OriginatorId: originatorId,
			TargetId:     actorId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
