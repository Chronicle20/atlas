package invite

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdStatusEventProvider(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &statusEvent[createdEventBody]{
		WorldId:     worldId,
		InviteType:  inviteType,
		ReferenceId: referenceId,
		Type:        EventInviteStatusTypeCreated,
		Body: createdEventBody{
			OriginatorId: originatorId,
			TargetId:     targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func acceptedStatusEventProvider(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &statusEvent[acceptedEventBody]{
		WorldId:     worldId,
		InviteType:  inviteType,
		ReferenceId: referenceId,
		Type:        EventInviteStatusTypeAccepted,
		Body: acceptedEventBody{
			OriginatorId: originatorId,
			TargetId:     targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func rejectedStatusEventProvider(referenceId uint32, worldId byte, inviteType string, originatorId uint32, targetId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(referenceId))
	value := &statusEvent[rejectedEventBody]{
		WorldId:     worldId,
		InviteType:  inviteType,
		ReferenceId: referenceId,
		Type:        EventInviteStatusTypeRejected,
		Body: rejectedEventBody{
			OriginatorId: originatorId,
			TargetId:     targetId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
