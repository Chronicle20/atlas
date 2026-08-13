package kite

import (
	kiteMsg "atlas-kites/kafka/message/kite"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// createdStatusEventProvider builds the KITE_CREATED status event, keyed on
// the field's map id for per-map ordering (mirroring the chalkboard/mist
// producers).
func createdStatusEventProvider(transactionId uuid.UUID, m Model) model.Provider[[]kafka.Message] {
	f := m.Field()
	key := producer.CreateKey(int(f.MapId()))
	value := &kiteMsg.StatusEvent[kiteMsg.CreatedStatusEventBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   m.CharacterId(),
		Type:          kiteMsg.EventTopicStatusTypeCreated,
		Body: kiteMsg.CreatedStatusEventBody{
			KiteId:     m.Id(),
			Name:       m.Name(),
			TemplateId: m.TemplateId(),
			Message:    m.Message(),
			X:          m.X(),
			Y:          m.Y(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// destroyedStatusEventProvider builds the KITE_DESTROYED status event. m must
// carry the field the kite was destroyed FROM -- captured by the caller
// before the registry removal -- so the event fans out to the right map.
func destroyedStatusEventProvider(transactionId uuid.UUID, m Model, reason string) model.Provider[[]kafka.Message] {
	f := m.Field()
	key := producer.CreateKey(int(f.MapId()))
	value := &kiteMsg.StatusEvent[kiteMsg.DestroyedStatusEventBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   m.CharacterId(),
		Type:          kiteMsg.EventTopicStatusTypeDestroyed,
		Body: kiteMsg.DestroyedStatusEventBody{
			KiteId: m.Id(),
			Reason: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// creationFailedStatusEventProvider builds the KITE_CREATION_FAILED status
// event. It targets a single character -- the envelope's CharacterId is the
// addressee, not a map broadcast -- because a refused placement is only ever
// interesting to the character who tried to place it.
func creationFailedStatusEventProvider(transactionId uuid.UUID, f field.Model, characterId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(f.MapId()))
	value := &kiteMsg.StatusEvent[kiteMsg.CreationFailedStatusEventBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		CharacterId:   characterId,
		Type:          kiteMsg.EventTopicStatusTypeCreationFailed,
		Body: kiteMsg.CreationFailedStatusEventBody{
			Reason: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
