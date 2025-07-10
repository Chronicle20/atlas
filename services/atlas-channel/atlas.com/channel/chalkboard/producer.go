package chalkboard

import (
	chalkboard2 "atlas-channel/kafka/message/chalkboard"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func SetCommandProvider(m _map.Model, characterId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &chalkboard2.Command[chalkboard2.SetCommandBody]{
		WorldId:     world.Id(m.WorldId()),
		ChannelId:   channel.Id(m.ChannelId()),
		MapId:       _map.Id(m.MapId()),
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
		WorldId:     world.Id(m.WorldId()),
		ChannelId:   channel.Id(m.ChannelId()),
		MapId:       _map.Id(m.MapId()),
		CharacterId: characterId,
		Type:        chalkboard2.CommandChalkboardClear,
		Body:        chalkboard2.ClearCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
