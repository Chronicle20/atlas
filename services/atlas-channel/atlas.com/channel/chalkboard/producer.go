package chalkboard

import (
	chalkboard2 "atlas-channel/kafka/message/chalkboard"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func SetCommandProvider(f field.Model, characterId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chalkboard2.Command[chalkboard2.SetCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        chalkboard2.CommandChalkboardSet,
		Body: chalkboard2.SetCommandBody{
			Message: message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ClearCommandProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chalkboard2.Command[chalkboard2.ClearCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        chalkboard2.CommandChalkboardClear,
		Body:        chalkboard2.ClearCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
