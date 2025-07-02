package invite

import (
	invite2 "atlas-invites/kafka/message/invite"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdStatusEventProvider(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &invite2.StatusEvent[invite2.CreatedEventBody]{
		WorldId:     worldId,
		InviteType:  inviteType,
		ReferenceId: referenceId,
		Type:        invite2.EventInviteStatusTypeCreated,
		Body: invite2.CreatedEventBody{
			OriginatorId: originatorId,
			TargetId:     targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func acceptedStatusEventProvider(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &invite2.StatusEvent[invite2.AcceptedEventBody]{
		WorldId:     worldId,
		InviteType:  inviteType,
		ReferenceId: referenceId,
		Type:        invite2.EventInviteStatusTypeAccepted,
		Body: invite2.AcceptedEventBody{
			OriginatorId: originatorId,
			TargetId:     targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func rejectedStatusEventProvider(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &invite2.StatusEvent[invite2.RejectedEventBody]{
		WorldId:     worldId,
		InviteType:  inviteType,
		ReferenceId: referenceId,
		Type:        invite2.EventInviteStatusTypeRejected,
		Body: invite2.RejectedEventBody{
			OriginatorId: originatorId,
			TargetId:     targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
