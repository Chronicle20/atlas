package chalkboard

import (
	chalkboard2 "atlas-chalkboards/kafka/message/chalkboard"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func setStatusEventProvider(transactionId uuid.UUID, field field.Model, characterId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(field.MapId()))
	value := &chalkboard2.StatusEvent[chalkboard2.SetStatusEventBody]{
		TransactionId: transactionId,
		WorldId:       field.WorldId(),
		ChannelId:     field.ChannelId(),
		MapId:         field.MapId(),
		CharacterId:   characterId,
		Type:          chalkboard2.EventTopicStatusTypeSet,
		Body: chalkboard2.SetStatusEventBody{
			Message: message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func clearStatusEventProvider(transactionId uuid.UUID, field field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(field.MapId()))
	value := &chalkboard2.StatusEvent[chalkboard2.ClearStatusEventBody]{
		TransactionId: transactionId,
		WorldId:       field.WorldId(),
		ChannelId:     field.ChannelId(),
		MapId:         field.MapId(),
		CharacterId:   characterId,
		Type:          chalkboard2.EventTopicStatusTypeClear,
		Body:          chalkboard2.ClearStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
