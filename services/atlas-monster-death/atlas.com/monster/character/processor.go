package character

import (
	"atlas-monster-death/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return requests.Provider[RestModel, Model](l, ctx)(requestById(characterId), Extract)()
		}
	}
}

func AwardExperience(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, white bool, amount uint32, party uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, characterId uint32, white bool, amount uint32, party uint32) error {
		return func(worldId byte, channelId byte, characterId uint32, white bool, amount uint32, party uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(awardExperienceCommandProvider(characterId, worldId, channelId, white, amount, party))
		}
	}
}
