package character

import (
	"atlas-monster-death/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/sirupsen/logrus"
)

func AwardExperience(l logrus.FieldLogger) func(ctx context.Context) func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
	return func(ctx context.Context) func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
		return func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(awardExperienceCommandProvider(characterId, ch, white, amount, party))
		}
	}
}
