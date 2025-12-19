package invite

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(actorId uint32, referenceId uint32, worldId byte, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &commandEvent[createCommandBody]{
		WorldId:    worldId,
		InviteType: InviteTypeGuild,
		Type:       CommandInviteTypeCreate,
		Body: createCommandBody{
			OriginatorId: actorId,
			TargetId:     targetId,
			ReferenceId:  referenceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
