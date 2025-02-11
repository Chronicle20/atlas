package character

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func ChangeMapProvider(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeMapBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        CommandCharacterChangeMap,
		Body: changeMapBody{
			ChannelId: channelId,
			MapId:     mapId,
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func awardExperienceCommandProvider(characterId uint32, worldId byte, channelId byte, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[awardExperienceCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        CommandAwardExperience,
		Body: awardExperienceCommandBody{
			ChannelId: channelId,
			Distributions: []experienceDistributions{{
				ExperienceType: ExperienceDistributionTypeWhite,
				Amount:         amount,
			}},
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func awardLevelCommandProvider(characterId uint32, worldId byte, channelId byte, amount byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[awardLevelCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        CommandAwardLevel,
		Body: awardLevelCommandBody{
			ChannelId: channelId,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeJobCommandProvider(characterId uint32, worldId byte, channelId byte, jobId uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeJobCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        CommandChangeJob,
		Body: changeJobCommandBody{
			ChannelId: channelId,
			JobId:     jobId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
