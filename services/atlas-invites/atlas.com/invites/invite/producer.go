package invite

import (
	invite2 "atlas-invites/kafka/message/invite"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func createdStatusEventProvider(referenceId uint32, worldId world.Id, inviteType string, originatorId uint32, targetId uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &invite2.StatusEvent[invite2.CreatedEventBody]{
		WorldId:       worldId,
		InviteType:    invite.Type(inviteType),
		ReferenceId:   invite.Id(referenceId),
		Type:          invite.StatusTypeCreated,
		TransactionId: transactionId,
		Body: invite2.CreatedEventBody{
			OriginatorId: character.Id(originatorId),
			TargetId:     character.Id(targetId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func acceptedStatusEventProvider(referenceId uint32, worldId world.Id, inviteType string, originatorId uint32, targetId uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &invite2.StatusEvent[invite2.AcceptedEventBody]{
		WorldId:       worldId,
		InviteType:    invite.Type(inviteType),
		ReferenceId:   invite.Id(referenceId),
		Type:          invite.StatusTypeAccepted,
		TransactionId: transactionId,
		Body: invite2.AcceptedEventBody{
			OriginatorId: character.Id(originatorId),
			TargetId:     character.Id(targetId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func rejectedStatusEventProvider(referenceId uint32, worldId world.Id, inviteType string, originatorId uint32, targetId uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &invite2.StatusEvent[invite2.RejectedEventBody]{
		WorldId:       worldId,
		InviteType:    invite.Type(inviteType),
		ReferenceId:   invite.Id(referenceId),
		Type:          invite.StatusTypeRejected,
		TransactionId: transactionId,
		Body: invite2.RejectedEventBody{
			OriginatorId: character.Id(originatorId),
			TargetId:     character.Id(targetId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
