package chalkboard

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func setCommandProvider(m _map.Model, characterId uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[setCommandBody]{
		WorldId:     byte(m.WorldId()),
		ChannelId:   byte(m.ChannelId()),
		MapId:       uint32(m.MapId()),
		CharacterId: characterId,
		Type:        CommandChalkboardSet,
		Body: setCommandBody{
			Message: message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func clearCommandProvider(m _map.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &commandEvent[clearCommandBody]{
		WorldId:     byte(m.WorldId()),
		ChannelId:   byte(m.ChannelId()),
		MapId:       uint32(m.MapId()),
		CharacterId: characterId,
		Type:        CommandChalkboardClear,
		Body:        clearCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
