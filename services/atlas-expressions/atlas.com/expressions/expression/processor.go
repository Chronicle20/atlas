package expression

import (
	"atlas-expressions/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func Change(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32, worldId byte, channelId byte, mapId uint32, expression uint32) error {
	return func(ctx context.Context) func(characterId uint32, worldId byte, channelId byte, mapId uint32, expression uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(characterId uint32, worldId byte, channelId byte, mapId uint32, expression uint32) error {
			l.Debugf("Changing expression to [%d] for character [%d] in character [%d].", expression, characterId, mapId)
			GetRegistry().add(t, characterId, worldId, channelId, mapId, expression)
			return producer.ProviderImpl(l)(ctx)(EnvExpressionEvent)(expressionEventProvider(characterId, worldId, channelId, mapId, expression))
		}
	}
}

func Clear(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) error {
	return func(ctx context.Context) func(characterId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(characterId uint32) error {
			l.Debugf("Clearing expression for character [%d].", characterId)
			GetRegistry().clear(t, characterId)
			return nil
		}
	}
}
