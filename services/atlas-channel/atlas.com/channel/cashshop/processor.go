package cashshop

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

func Enter(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId world.Id) error {
	return func(ctx context.Context) func(characterId uint32, worldId world.Id) error {
		return func(characterId uint32, worldId world.Id) error {
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicCashShopStatus)(characterEnterCashShopStatusEventProvider(characterId, worldId))
		}
	}
}

func Exit(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId world.Id) error {
	return func(ctx context.Context) func(characterId uint32, worldId world.Id) error {
		return func(characterId uint32, worldId world.Id) error {
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicCashShopStatus)(characterExitCashShopStatusEventProvider(characterId, worldId))
		}
	}
}
