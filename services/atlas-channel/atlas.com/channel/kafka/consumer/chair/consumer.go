package chair

import (
	"atlas-channel/chair"
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("chair_status_event")(EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventUsed(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCancelled(sc, wp))))
			}
		}
	}
}

func handleStatusEventUsed(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventUsedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventUsedBody]) {
		if e.Type != EventStatusTypeUsed {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		if e.ChairType == chair.ChairTypePortable {
			_ = _map.ForOtherSessionsInMap(l)(ctx)(sc.WorldId(), sc.ChannelId(), e.MapId, e.Body.CharacterId, showChair(l)(ctx)(wp)(e.Body.CharacterId, e.ChairId))

			session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, func(s session.Model) error {
				err := session.Announce(l)(ctx)(wp)(writer.StatChanged)(s, writer.StatChangedBody(l)(make([]model2.StatUpdate, 0), true))
				if err != nil {
					l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
					return err
				}
				return nil
			})
			return
		}
		if e.ChairType == chair.ChairTypeFixed {
			session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, func(s session.Model) error {
				err := session.Announce(l)(ctx)(wp)(writer.CharacterSitResult)(s, writer.CharacterSitBody(uint16(e.ChairId)))
				if err != nil {
					l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.CharacterSitResult, s.CharacterId())
					return err
				}
				return nil
			})
			return
		}
	}
}

func showChair(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, chairId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, chairId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, chairId uint32) model.Operator[session.Model] {
			return func(characterId uint32, chairId uint32) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := session.Announce(l)(ctx)(wp)(writer.CharacterShowChair)(s, writer.CharacterShowChairBody(characterId, chairId))
					if err != nil {
						return err
					}
					return nil
				}
			}
		}
	}
}

func handleStatusEventCancelled(sc server.Model, wp writer.Producer) message.Handler[statusEvent[statusEventCancelledBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[statusEventCancelledBody]) {
		if e.Type != EventStatusTypeCancelled {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		_ = _map.ForOtherSessionsInMap(l)(ctx)(sc.WorldId(), sc.ChannelId(), e.MapId, e.Body.CharacterId, showChair(l)(ctx)(wp)(e.Body.CharacterId, 0))

		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, func(s session.Model) error {
			err := session.Announce(l)(ctx)(wp)(writer.CharacterSitResult)(s, writer.CharacterCancelSitBody())
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.CharacterSitResult, s.CharacterId())
				return err
			}
			return nil
		})
	}
}
