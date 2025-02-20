package expression

import (
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

func Change(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, m _map.Model, expression uint32) error {
	return func(ctx context.Context) func(characterId uint32, m _map.Model, expression uint32) error {
		return func(characterId uint32, m _map.Model, expression uint32) error {
			l.Debugf("Changing character [%d] expression to [%d].", characterId, m.MapId())
			return producer.ProviderImpl(l)(ctx)(EnvExpressionCommand)(expressionCommandProvider(characterId, m, expression))
		}
	}
}
