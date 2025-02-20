package fame

import (
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

func RequestChange(l logrus.FieldLogger) func(ctx context.Context) func(m _map.Model, characterId uint32, targetId uint32, amount int8) error {
	return func(ctx context.Context) func(m _map.Model, characterId uint32, targetId uint32, amount int8) error {
		return func(m _map.Model, characterId uint32, targetId uint32, amount int8) error {
			return producer.ProviderImpl(l)(ctx)(EnvCommandTopic)(requestChangeFameCommandProvider(m, characterId, targetId, amount))
		}
	}
}
