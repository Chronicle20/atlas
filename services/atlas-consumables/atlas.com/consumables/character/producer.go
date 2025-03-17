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

func changeMapProvider(m _map.Model, characterId uint32, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeMapBody]{
		WorldId:     byte(m.WorldId()),
		CharacterId: characterId,
		Type:        CommandChangeMap,
		Body: changeMapBody{
			ChannelId: byte(m.ChannelId()),
			MapId:     uint32(m.MapId()),
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
