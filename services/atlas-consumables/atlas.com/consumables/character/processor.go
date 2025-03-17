package character

import (
	"atlas-consumables/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func ChangeHP(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, amount int16) error {
	return func(ctx context.Context) func(characterId uint32, amount int16) error {
		return func(characterId uint32, amount int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeHPCommandProvider(characterId, amount))
		}
	}
}

func ChangeMP(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, amount int16) error {
	return func(ctx context.Context) func(characterId uint32, amount int16) error {
		return func(characterId uint32, amount int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeMPCommandProvider(characterId, amount))
		}
	}
}
