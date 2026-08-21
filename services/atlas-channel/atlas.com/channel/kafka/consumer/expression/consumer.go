package expression

import (
	consumer2 "atlas-channel/kafka/consumer"
	expression2 "atlas-channel/kafka/message/expression"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("expression_event")(expression2.EnvExpressionEvent)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(expression2.EnvExpressionEvent)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleEvent(sc server.Model, wp writer.Producer) message.Handler[expression2.Event] {
	return func(l logrus.FieldLogger, ctx context.Context, e expression2.Event) {
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		// e.Duration is int32 and is -1 for every emote a GMS v95 client
		// originates (CWvsContext::SendEmotionChange@0xa02c86 and
		// CUserLocal::UseFuncKeyMapped case 3u@0x933874 both pass nDuration
		// = -1). Go's int32 -> uint32 conversion preserves the bit pattern, so
		// -1 reaches the wire as FF FF FF FF — exactly what the sending client
		// encoded. This is deliberate, not a lost sign.
		err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Field(e.MapId, e.Instance), e.CharacterId, session.Announce(l)(ctx)(wp)(charpkt.CharacterExpressionWriter)(charpkt.NewCharacterExpression(e.CharacterId, e.Expression, uint32(e.Duration), e.ByItemOption).Encode))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce character [%d] expression [%d] change to characters in map [%d].", e.CharacterId, e.Expression, e.MapId)
		}
	}
}
