package expression

import (
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const consumerEvent = "expression_event"

func EventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerEvent)(EnvExpressionEvent)(groupId)
	}
}

func EventRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvExpressionEvent)()
		return t, message.AdaptHandler(message.PersistentConfig(handleEvent(sc, wp)))
	}
}

func handleEvent(sc server.Model, wp writer.Producer) message.Handler[expressionEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e expressionEvent) {
		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != e.WorldId {
			return
		}
		if sc.ChannelId() != e.ChannelId {
			return
		}
		_ = _map.ForOtherSessionsInMap(l)(ctx)(e.WorldId, e.ChannelId, e.MapId, e.CharacterId, func(s session.Model) error {
			characterExpressionFunc := session.Announce(l)(ctx)(wp)(writer.CharacterExpression)
			err := characterExpressionFunc(s, writer.CharacterExpressionBody(e.CharacterId, e.Expression))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce character [%d] expression [%d] change to [%d].", e.CharacterId, e.Expression, s.CharacterId())
				return err
			}
			return nil
		})

	}
}
