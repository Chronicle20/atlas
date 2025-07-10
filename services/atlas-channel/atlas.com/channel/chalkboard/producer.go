package chalkboard

import (
	chalkboard2 "atlas-channel/kafka/message/chalkboard"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func SetCommandProvider(m _map.Model, characterId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chalkboard2.Command[chalkboard2.SetCommandBody]{
		WorldId:     m.WorldId(),
		ChannelId:   m.ChannelId(),
		MapId:       m.MapId(),
		CharacterId: characterId,
		Type:        chalkboard2.CommandChalkboardSet,
		Body: chalkboard2.SetCommandBody{
			Message: message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ClearCommandProvider(m _map.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chalkboard2.Command[chalkboard2.ClearCommandBody]{
		WorldId:     m.WorldId(),
		ChannelId:   m.ChannelId(),
		MapId:       m.MapId(),
		CharacterId: characterId,
		Type:        chalkboard2.CommandChalkboardClear,
		Body:        chalkboard2.ClearCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
