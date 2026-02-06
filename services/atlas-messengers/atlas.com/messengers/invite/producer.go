package invite

import (
	invite2 "atlas-messengers/kafka/message/invite"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(transactionID uuid.UUID, actorId uint32, messengerId uint32, worldId world.Id, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &invite2.Command[invite2.CreateCommandBody]{
		TransactionId: transactionID,
		WorldId:       worldId,
		InviteType:    invite.TypeMessenger,
		Type:          invite.CommandTypeCreate,
		Body: invite2.CreateCommandBody{
			OriginatorId: character.Id(actorId),
			TargetId:     character.Id(targetId),
			ReferenceId:  invite.Id(messengerId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
