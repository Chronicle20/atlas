package invite

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(actorId uint32, partyId uint32, worldId byte, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &commandEvent[createCommandBody]{
		WorldId: worldId,
		Type:    CommandInviteTypeCreate,
		Body: createCommandBody{
			InviteType:   InviteTypeParty,
			OriginatorId: actorId,
			TargetId:     targetId,
			ReferenceId:  partyId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
