package character

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func changeHPCommandProvider(m _map.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeHPCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        CommandChangeHP,
		Body: changeHPCommandBody{
			ChannelId: byte(m.ChannelId()),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMPCommandProvider(m _map.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeMPCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        CommandChangeMP,
		Body: changeMPCommandBody{
			ChannelId: byte(m.ChannelId()),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
