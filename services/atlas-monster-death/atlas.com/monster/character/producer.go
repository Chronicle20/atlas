package character

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func awardExperienceCommandProvider(characterId uint32, worldId world.Id, channelId channel.Id, white bool, amount uint32, party uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	ds := make([]experienceDistributions, 0)
	if white {
		ds = append(ds, experienceDistributions{
			ExperienceType: ExperienceDistributionTypeWhite,
			Amount:         amount,
		})
	} else {
		ds = append(ds, experienceDistributions{
			ExperienceType: ExperienceDistributionTypeYellow,
			Amount:         amount,
		})
	}
	ds = append(ds, experienceDistributions{
		ExperienceType: ExperienceDistributionTypeParty,
		Amount:         party,
	})

	value := &command[awardExperienceCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        CommandAwardExperience,
		Body: awardExperienceCommandBody{
			ChannelId:     channelId,
			Distributions: ds,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
