package cashshop

import (
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

func Enter(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, m _map.Model) error {
	return func(ctx context.Context) func(characterId uint32, m _map.Model) error {
		return func(characterId uint32, m _map.Model) error {
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicCashShopStatus)(characterEnterCashShopStatusEventProvider(characterId, m))
		}
	}
}

func Exit(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, m _map.Model) error {
	return func(ctx context.Context) func(characterId uint32, m _map.Model) error {
		return func(characterId uint32, m _map.Model) error {
			return producer.ProviderImpl(l)(ctx)(EnvEventTopicCashShopStatus)(characterExitCashShopStatusEventProvider(characterId, m))
		}
	}
}
