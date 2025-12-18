package character

import (
	character2 "atlas-consumables/kafka/message/character"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func changeHPCommandProvider(m _map.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeHPCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        character2.CommandChangeHP,
		Body: character2.ChangeHPCommandBody{
			ChannelId: byte(m.ChannelId()),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMPCommandProvider(m _map.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMPCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        character2.CommandChangeMP,
		Body: character2.ChangeMPCommandBody{
			ChannelId: byte(m.ChannelId()),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMapProvider(m _map.Model, characterId uint32, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMapBody]{
		WorldId:     byte(m.WorldId()),
		CharacterId: characterId,
		Type:        character2.CommandChangeMap,
		Body: character2.ChangeMapBody{
			ChannelId: byte(m.ChannelId()),
			MapId:     uint32(m.MapId()),
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
