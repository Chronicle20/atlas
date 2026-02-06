package character

import (
	character2 "atlas-consumables/kafka/message/character"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func changeHPCommandProvider(f field.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeHPCommandBody]{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		Type:        character2.CommandChangeHP,
		Body: character2.ChangeHPCommandBody{
			ChannelId: f.ChannelId(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMPCommandProvider(f field.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMPCommandBody]{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		Type:        character2.CommandChangeMP,
		Body: character2.ChangeMPCommandBody{
			ChannelId: f.ChannelId(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMapProvider(f field.Model, characterId uint32, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMapBody]{
		WorldId:     f.WorldId(),
		CharacterId: characterId,
		Type:        character2.CommandChangeMap,
		Body: character2.ChangeMapBody{
			ChannelId: f.ChannelId(),
			MapId:     f.MapId(),
			Instance:  f.Instance(),
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
