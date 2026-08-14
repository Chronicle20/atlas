package pending_change

import (
	pendingchange2 "atlas-character/kafka/message/pending_change"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func createdEventProvider(m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.CharacterId()))
	value := &pendingchange2.StatusEvent[pendingchange2.CreatedEventBody]{
		TransactionId: m.TransactionId(),
		CharacterId:   m.CharacterId(),
		WorldId:       m.SourceWorldId(),
		Type:          pendingchange2.EventTypeCreated,
		Body: pendingchange2.CreatedEventBody{
			PendingChangeId:    m.Id(),
			ChangeType:         m.Type(),
			RequestedName:      m.RequestedName(),
			DestinationWorldId: m.DestinationWorldId(),
			ExpiresAt:          m.ExpiresAt(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func resolvedEventProvider(m Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(m.CharacterId()))
	value := &pendingchange2.StatusEvent[pendingchange2.ResolvedEventBody]{
		TransactionId: m.TransactionId(),
		CharacterId:   m.CharacterId(),
		WorldId:       m.SourceWorldId(),
		Type:          pendingchange2.EventTypeResolved,
		Body: pendingchange2.ResolvedEventBody{
			PendingChangeId:    m.Id(),
			ChangeType:         m.Type(),
			Status:             m.Status(),
			Reason:             m.Reason(),
			RequestedName:      m.RequestedName(),
			DestinationWorldId: m.DestinationWorldId(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
