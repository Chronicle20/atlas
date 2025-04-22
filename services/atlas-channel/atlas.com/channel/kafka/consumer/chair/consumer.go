package chair

import (
	consumer2 "atlas-channel/kafka/consumer"
	chair2 "atlas-channel/kafka/message/chair"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("chair_status_event")(chair2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(chair2.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventUsed(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventError(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCancelled(sc, wp))))
			}
		}
	}
}

func handleStatusEventUsed(sc server.Model, wp writer.Producer) message.Handler[chair2.StatusEvent[chair2.StatusEventUsedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e chair2.StatusEvent[chair2.StatusEventUsedBody]) {
		if e.Type != chair2.EventStatusTypeUsed {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		if e.ChairType == chair2.TypePortable {
			err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Map(_map2.Id(e.MapId)), e.Body.CharacterId, showChairForeign(l)(ctx)(wp)(e.Body.CharacterId, e.ChairId))
			if err != nil {
				l.WithError(err).Errorf("Unable to show [%d] using chair [%d] to those in map [%d].", e.Body.CharacterId, e.ChairId, e.MapId)
			}

			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, enableActions(l)(ctx)(wp))
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, e.Body.CharacterId)
			}
			return
		}
		if e.ChairType == chair2.TypeFixed {
			err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, showChair(l)(ctx)(wp)(e.ChairId))
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.CharacterSitResult, e.Body.CharacterId)
			}
			return
		}
	}
}

func showChair(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(chairId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(chairId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(chairId uint32) model.Operator[session.Model] {
			return func(chairId uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterSitResult)(writer.CharacterSitBody(uint16(chairId)))
			}
		}
	}
}

func showChairForeign(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, chairId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, chairId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, chairId uint32) model.Operator[session.Model] {
			return func(characterId uint32, chairId uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterShowChair)(writer.CharacterShowChairBody(characterId, chairId))
			}
		}
	}
}

func handleStatusEventError(sc server.Model, wp writer.Producer) message.Handler[chair2.StatusEvent[chair2.StatusEventErrorBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e chair2.StatusEvent[chair2.StatusEventErrorBody]) {
		if e.Type != chair2.EventStatusTypeError {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		if e.Body.Type != chair2.ErrorTypeInternal {
			_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, session.NewProcessor(l, ctx).Destroy)
			l.Errorf("Character [%d] attempting to perform chair action they cannot.", e.Body.CharacterId)
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, enableActions(l)(ctx)(wp))
		l.Warnf("Internal issue performing character [%d] sit action.", e.Body.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, e.Body.CharacterId)
		}
		return
	}
}

func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
		return func(wp writer.Producer) func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)(make([]model2.StatUpdate, 0), true))
		}
	}
}

func handleStatusEventCancelled(sc server.Model, wp writer.Producer) message.Handler[chair2.StatusEvent[chair2.StatusEventCancelledBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e chair2.StatusEvent[chair2.StatusEventCancelledBody]) {
		if e.Type != chair2.EventStatusTypeCancelled {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.Body.CharacterId, cancelChair(l)(ctx)(wp))
		if err != nil {
			l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.CharacterSitResult, e.Body.CharacterId)
		}

		err = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Map(_map2.Id(e.MapId)), e.Body.CharacterId, cancelChairForeign(l)(ctx)(wp)(e.Body.CharacterId))
		if err != nil {
			l.WithError(err).Errorf("Unable to write foreign character [%d] sit result completely to [%d].", e.Body.CharacterId, e.MapId)
		}
	}
}

func cancelChair(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) model.Operator[session.Model] {
		return func(wp writer.Producer) model.Operator[session.Model] {
			return session.Announce(l)(ctx)(wp)(writer.CharacterSitResult)(writer.CharacterCancelSitBody())
		}
	}
}

func cancelChairForeign(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32) model.Operator[session.Model] {
			return func(characterId uint32) model.Operator[session.Model] {
				return showChairForeign(l)(ctx)(wp)(characterId, 0)
			}
		}
	}
}
