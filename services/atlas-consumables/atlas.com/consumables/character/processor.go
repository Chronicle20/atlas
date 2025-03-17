package character

import (
	"atlas-consumables/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

func ChangeHP(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
		return func(m _map.Model, characterId uint32, amount int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeHPCommandProvider(m, characterId, amount))
		}
	}
}

func ChangeMP(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, amount int16) error {
		return func(m _map.Model, characterId uint32, amount int16) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(changeMPCommandProvider(m, characterId, amount))
		}
	}
}
