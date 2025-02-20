package chair

import (
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

func Use(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, chairType string, chairId uint32, characterId uint32) error {
	return func(ctx context.Context) func(m _map.Model, chairType string, chairId uint32, characterId uint32) error {
		return func(m _map.Model, chairType string, chairId uint32, characterId uint32) error {
			l.Debugf("Character [%d] attempting to use map [%d] [%s] chair [%d].", characterId, m.MapId(), chairType, chairId)
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(commandUseProvider(m, chairType, chairId, characterId))
		}
	}
}

func Cancel(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32) error {
		return func(m _map.Model, characterId uint32) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(commandCancelProvider(m, characterId))
		}
	}
}
