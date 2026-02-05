package invite

import (
	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(actorId uint32, referenceId uint32, worldId world.Id, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &commandEvent[createCommandBody]{
		WorldId:    worldId,
		InviteType: invite.TypeGuild,
		Type:       invite.CommandTypeCreate,
		Body: createCommandBody{
			OriginatorId: character.Id(actorId),
			TargetId:     character.Id(targetId),
			ReferenceId:  invite.Id(referenceId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
