package character

import (
	"atlas-channel/kafka/message/character"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func RequestDistributeApCommandProvider(f field.Model, characterId uint32, distributions []character.DistributePair) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character.Command[character.RequestDistributeApCommandBody]{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		Type:        character.CommandRequestDistributeAp,
		Body: character.RequestDistributeApCommandBody{
			Distributions: distributions,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestDistributeSpCommandProvider(f field.Model, characterId uint32, skillId uint32, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character.Command[character.RequestDistributeSpCommandBody]{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		Type:        character.CommandRequestDistributeSp,
		Body: character.RequestDistributeSpCommandBody{
			SkillId: skillId,
			Amount:  amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestDropMesoCommandProvider(f field.Model, characterId uint32, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character.Command[character.RequestDropMesoCommandBody]{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		Type:        character.CommandRequestDropMeso,
		Body: character.RequestDropMesoCommandBody{
			ChannelId: f.ChannelId(),
			MapId:     f.MapId(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeHPCommandProvider(f field.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character.Command[character.ChangeHPCommandBody]{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		Type:        character.CommandChangeHP,
		Body: character.ChangeHPCommandBody{
			ChannelId: f.ChannelId(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ChangeMPCommandProvider(f field.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character.Command[character.ChangeMPCommandBody]{
		CharacterId: characterId,
		WorldId:     f.WorldId(),
		Type:        character.CommandChangeMP,
		Body: character.ChangeMPCommandBody{
			ChannelId: f.ChannelId(),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
