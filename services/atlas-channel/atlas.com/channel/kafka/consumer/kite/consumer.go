package kite

import (
	consumer2 "atlas-channel/kafka/consumer"
	kiteMsg "atlas-channel/kafka/message/kite"
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
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("kite_status_event")(kiteMsg.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var err error
				var handles []listener.HandlerHandle
				t, err = topic.EnvProvider(l)(kiteMsg.EnvEventTopicStatus)()
				if err != nil {
					return nil, err
				}
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCreatedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleDestroyedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleCreationFailedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleCreatedEvent(sc server.Model, wp writer.Producer) message.Handler[kiteMsg.StatusEvent[kiteMsg.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e kiteMsg.StatusEvent[kiteMsg.CreatedStatusEventBody]) {
		if e.Type != kiteMsg.EventTopicStatusTypeCreated {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(fieldcb.KiteSpawnWriter)(
				fieldcb.NewKiteSpawn(e.Body.KiteId, e.Body.TemplateId, e.Body.Message, e.Body.Name, e.Body.X, e.Body.Y).Encode)(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn kite [%d] for character [%d].", e.Body.KiteId, e.CharacterId)
		}
	}
}

func handleDestroyedEvent(sc server.Model, wp writer.Producer) message.Handler[kiteMsg.StatusEvent[kiteMsg.DestroyedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e kiteMsg.StatusEvent[kiteMsg.DestroyedStatusEventBody]) {
		if e.Type != kiteMsg.EventTopicStatusTypeDestroyed {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		// KiteDestroyAnimated (0) plays the one-shot despawn animation. The
		// byte is a suppress-animation flag, not a selector; both destroy
		// reasons are the same class of event and want the animation.
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(fieldcb.KiteDestroyWriter)(
				fieldcb.NewKiteDestroy(e.Body.KiteId, fieldcb.KiteDestroyAnimated).Encode)(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to destroy kite [%d].", e.Body.KiteId)
		}
	}
}

func handleCreationFailedEvent(sc server.Model, wp writer.Producer) message.Handler[kiteMsg.StatusEvent[kiteMsg.CreationFailedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e kiteMsg.StatusEvent[kiteMsg.CreationFailedStatusEventBody]) {
		if e.Type != kiteMsg.EventTopicStatusTypeCreationFailed {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		// Targeted, NOT a map broadcast. FieldKiteError has an empty body, so
		// the client shows a generic failure and the reason survives only here.
		l.Infof("Kite placement refused for character [%d]: [%s].", e.CharacterId, e.Body.Reason)
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(fieldcb.KiteErrorWriter)(fieldcb.NewKiteError().Encode)(s)
		})
		if err != nil {
			l.WithError(err).Errorf("Unable to notify character [%d] of kite failure.", e.CharacterId)
		}
	}
}
