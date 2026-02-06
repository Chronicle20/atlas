package chair

import (
	chair2 "atlas-chairs/kafka/message/chair"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func statusEventUsedProvider(field field.Model, chairType string, chairId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chair2.StatusEvent[chair2.StatusEventUsedBody]{
		WorldId:   field.WorldId(),
		ChannelId: field.ChannelId(),
		MapId:     field.MapId(),
		ChairType: chairType,
		ChairId:   chairId,
		Type:      chair2.EventStatusTypeUsed,
		Body: chair2.StatusEventUsedBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventErrorProvider(field field.Model, chairType string, chairId uint32, characterId uint32, errorType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chair2.StatusEvent[chair2.StatusEventErrorBody]{
		WorldId:   field.WorldId(),
		ChannelId: field.ChannelId(),
		MapId:     field.MapId(),
		ChairType: chairType,
		ChairId:   chairId,
		Type:      chair2.EventStatusTypeError,
		Body: chair2.StatusEventErrorBody{
			CharacterId: characterId,
			Type:        errorType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statusEventCancelledProvider(field field.Model, chairType string, chairId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chair2.StatusEvent[chair2.StatusEventCancelledBody]{
		WorldId:   field.WorldId(),
		ChannelId: field.ChannelId(),
		MapId:     field.MapId(),
		ChairType: chairType,
		ChairId:   chairId,
		Type:      chair2.EventStatusTypeCancelled,
		Body: chair2.StatusEventCancelledBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
