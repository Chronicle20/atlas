package character

import (
	character2 "atlas-messages/kafka/message/character"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func ChangeMapProvider(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMapBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        character2.CommandCharacterChangeMap,
		Body: character2.ChangeMapBody{
			ChannelId: channelId,
			MapId:     mapId,
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func awardExperienceCommandProvider(characterId uint32, worldId byte, channelId byte, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.AwardExperienceCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        character2.CommandAwardExperience,
		Body: character2.AwardExperienceCommandBody{
			ChannelId: channelId,
			Distributions: []character2.ExperienceDistributions{{
				ExperienceType: character2.ExperienceDistributionTypeWhite,
				Amount:         amount,
			}},
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func awardLevelCommandProvider(characterId uint32, worldId byte, channelId byte, amount byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.AwardLevelCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        character2.CommandAwardLevel,
		Body: character2.AwardLevelCommandBody{
			ChannelId: channelId,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeJobCommandProvider(characterId uint32, worldId byte, channelId byte, jobId uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeJobCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        character2.CommandChangeJob,
		Body: character2.ChangeJobCommandBody{
			ChannelId: channelId,
			JobId:     jobId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestChangeMesoCommandProvider(characterId uint32, worldId byte, actorId uint32, actorType string, amount int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.RequestChangeMesoBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        character2.CommandRequestChangeMeso,
		Body: character2.RequestChangeMesoBody{
			ActorId:   actorId,
			ActorType: actorType,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
