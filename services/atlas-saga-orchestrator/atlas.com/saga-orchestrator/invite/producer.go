package invite

import (
	invite2 "atlas-saga-orchestrator/kafka/message/invite"

	"github.com/Chronicle20/atlas-constants/character"
	"github.com/Chronicle20/atlas-constants/invite"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(transactionId uuid.UUID, inviteType string, actorId uint32, referenceId uint32, worldId world.Id, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &invite2.Command[invite2.CreateCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		InviteType:    invite.Type(inviteType),
		Type:          invite.CommandTypeCreate,
		Body: invite2.CreateCommandBody{
			OriginatorId: character.Id(actorId),
			TargetId:     character.Id(targetId),
			ReferenceId:  invite.Id(referenceId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func acceptInviteCommandProvider(transactionId uuid.UUID, inviteType string, worldId world.Id, referenceId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &invite2.Command[invite2.AcceptCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		InviteType:    invite.Type(inviteType),
		Type:          invite.CommandTypeAccept,
		Body: invite2.AcceptCommandBody{
			TargetId:    character.Id(targetId),
			ReferenceId: invite.Id(referenceId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func rejectInviteCommandProvider(transactionId uuid.UUID, inviteType string, worldId world.Id, originatorId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(targetId))
	value := &invite2.Command[invite2.RejectCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		InviteType:    invite.Type(inviteType),
		Type:          invite.CommandTypeReject,
		Body: invite2.RejectCommandBody{
			TargetId:     character.Id(targetId),
			OriginatorId: character.Id(originatorId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
