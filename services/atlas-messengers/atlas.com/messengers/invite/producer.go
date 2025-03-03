package invite

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(actorId uint32, messengerId uint32, worldId byte, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &commandEvent[createCommandBody]{
		WorldId:    worldId,
		InviteType: InviteTypeMessenger,
		Type:       CommandInviteTypeCreate,
		Body: createCommandBody{
			OriginatorId: actorId,
			TargetId:     targetId,
			ReferenceId:  messengerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
