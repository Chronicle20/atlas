package expression

import (
	"atlas-channel/kafka/producer"
	"context"
	"github.com/sirupsen/logrus"
)

func Change(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId byte, channelId byte, mapId uint32, expression uint32) error {
	return func(ctx context.Context) func(characterId uint32, worldId byte, channelId byte, mapId uint32, expression uint32) error {
		return func(characterId uint32, worldId byte, channelId byte, mapId uint32, expression uint32) error {
			l.Debugf("Changing character [%d] expression to [%d].", characterId, mapId)
			return producer.ProviderImpl(l)(ctx)(EnvExpressionCommand)(expressionCommandProvider(characterId, worldId, channelId, mapId, expression))
		}
	}
}
