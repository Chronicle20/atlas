package character

import (
	"atlas-monster-death/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

func AwardExperience(l logrus.FieldLogger) func(ctx context.Context) func(worldId world.Id, channelId channel.Id, characterId uint32, white bool, amount uint32, party uint32) error {
	return func(ctx context.Context) func(worldId world.Id, channelId channel.Id, characterId uint32, white bool, amount uint32, party uint32) error {
		return func(worldId world.Id, channelId channel.Id, characterId uint32, white bool, amount uint32, party uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(awardExperienceCommandProvider(characterId, worldId, channelId, white, amount, party))
		}
	}
}
