package invite

import (
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(actorId uint32, partyId uint32, worldId world.Id, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &commandEvent[createCommandBody]{
		WorldId:    worldId,
		InviteType: invite.TypeParty,
		Type:       invite.CommandTypeCreate,
		Body: createCommandBody{
			OriginatorId: character.Id(actorId),
			TargetId:     character.Id(targetId),
			ReferenceId:  invite.Id(partyId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
