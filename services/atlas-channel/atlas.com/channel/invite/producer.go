package invite

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func rejectInviteCommandProvider(actorId uint32, worldId byte, inviteType string, originatorId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &commandEvent[rejectCommandBody]{
		WorldId:    worldId,
		InviteType: inviteType,
		Type:       CommandInviteTypeReject,
		Body: rejectCommandBody{
			OriginatorId: originatorId,
			TargetId:     actorId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
