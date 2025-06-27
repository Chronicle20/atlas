package character

import (
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func ChangeMapProvider(transactionId uuid.UUID, characterId uint32, field field.Model, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMapBody]{
		TransactionId: transactionId,
		WorldId:       field.WorldId(),
		CharacterId:   characterId,
		Type:          character2.CommandChangeMap,
		Body: character2.ChangeMapBody{
			ChannelId: field.ChannelId(),
			MapId:     field.MapId(),
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AwardExperienceProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, distributions []character2.ExperienceDistributions) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.AwardExperienceCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          character2.CommandAwardExperience,
		Body: character2.AwardExperienceCommandBody{
			ChannelId:     channelId,
			Distributions: distributions,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AwardLevelProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, amount byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.AwardLevelCommandBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          character2.CommandAwardLevel,
		Body: character2.AwardLevelCommandBody{
			ChannelId: channelId,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AwardMesosProvider(transactionId uuid.UUID, worldId world.Id, characterId uint32, channelId channel.Id, actorId uint32, actorType string, amount int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.RequestChangeMesoBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		CharacterId:   characterId,
		Type:          character2.CommandRequestChangeMeso,
		Body: character2.RequestChangeMesoBody{
			ActorId:   actorId,
			ActorType: actorType,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
