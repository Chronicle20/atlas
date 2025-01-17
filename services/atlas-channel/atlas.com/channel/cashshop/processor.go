package cashshop

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func Enter(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId byte) error {
	return func(ctx context.Context) func(characterId uint32, worldId byte) error {
		return func(characterId uint32, worldId byte) error {
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicCashShopStatus)(characterEnterCashShopStatusEventProvider(characterId, worldId))
		}
	}
}

func Exit(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId byte) error {
	return func(ctx context.Context) func(characterId uint32, worldId byte) error {
		return func(characterId uint32, worldId byte) error {
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicCashShopStatus)(characterExitCashShopStatusEventProvider(characterId, worldId))
		}
	}
}
