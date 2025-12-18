package invite

import (
	"atlas-messengers/kafka/message/invite"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(transactionID uuid.UUID, actorId uint32, messengerId uint32, worldId world.Id, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(messengerId))
	value := &invite.CommandEvent[invite.CreateCommandBody]{
		TransactionID: transactionID,
		WorldId:       worldId,
		InviteType:    invite.InviteTypeMessenger,
		Type:          invite.CommandInviteTypeCreate,
		Body: invite.CreateCommandBody{
			OriginatorId: actorId,
			TargetId:     targetId,
			ReferenceId:  messengerId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
