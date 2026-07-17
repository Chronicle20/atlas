package chair

import (
	chair2 "atlas-chairs/kafka/message/chair"
	character2 "atlas-chairs/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func changeHPCommandProvider(field field.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeHPCommandBody]{
		WorldId:     field.WorldId(),
		CharacterId: characterId,
		Type:        character2.CommandChangeHP,
		Body: character2.ChangeHPCommandBody{
			ChannelId: field.ChannelId(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMPCommandProvider(field field.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMPCommandBody]{
		WorldId:     field.WorldId(),
		CharacterId: characterId,
		Type:        character2.CommandChangeMP,
		Body: character2.ChangeMPCommandBody{
			ChannelId: field.ChannelId(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
