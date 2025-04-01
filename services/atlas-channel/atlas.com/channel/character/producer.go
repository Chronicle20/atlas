package character

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestDistributeApCommandProvider(m _map.Model, characterId uint32, distributions []DistributePair) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestDistributeApCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        CommandRequestDistributeAp,
		Body: requestDistributeApCommandBody{
			Distributions: distributions,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestDistributeSpCommandProvider(m _map.Model, characterId uint32, skillId uint32, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestDistributeSpCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        CommandRequestDistributeSp,
		Body: requestDistributeSpCommandBody{
			SkillId: skillId,
			Amount:  amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestDropMesoCommandProvider(m _map.Model, characterId uint32, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestDropMesoCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        CommandRequestDropMeso,
		Body: requestDropMesoCommandBody{
			ChannelId: byte(m.ChannelId()),
			MapId:     uint32(m.MapId()),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

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
