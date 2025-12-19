package invite

import (
	"atlas-saga-orchestrator/kafka/message/invite"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func createInviteCommandProvider(transactionId uuid.UUID, inviteType string, actorId uint32, referenceId uint32, worldId byte, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &invite.CommandEvent[invite.CreateCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		InviteType:    inviteType,
		Type:          invite.CommandInviteTypeCreate,
		Body: invite.CreateCommandBody{
			OriginatorId: actorId,
			TargetId:     targetId,
			ReferenceId:  referenceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func acceptInviteCommandProvider(transactionId uuid.UUID, inviteType string, worldId byte, referenceId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &invite.CommandEvent[invite.AcceptCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		InviteType:    inviteType,
		Type:          invite.CommandInviteTypeAccept,
		Body: invite.AcceptCommandBody{
			TargetId:    targetId,
			ReferenceId: referenceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func rejectInviteCommandProvider(transactionId uuid.UUID, inviteType string, worldId byte, originatorId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(targetId))
	value := &invite.CommandEvent[invite.RejectCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		InviteType:    inviteType,
		Type:          invite.CommandInviteTypeReject,
		Body: invite.RejectCommandBody{
			TargetId:     targetId,
			OriginatorId: originatorId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
