package messenger

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	messenger2 "atlas-channel/kafka/message/messenger"
	"atlas-channel/messenger"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

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
			rf(consumer2.NewConfig(l)("messenger_status_event")(messenger2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(messenger2.EnvEventStatusTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLeft(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleJoin(sc, wp))))
			}
		}
	}
}

func handleLeft(sc server.Model, wp writer.Producer) message.Handler[messenger2.StatusEvent[messenger2.LeftEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e messenger2.StatusEvent[messenger2.LeftEventBody]) {
		if e.Type != messenger2.EventMessengerStatusTypeLeft {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := messenger.NewProcessor(l, ctx).GetById(e.MessengerId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for messenger [%d] which does not exist.", e.MessengerId)
			return
		}

		tc, err := character.NewProcessor(l, ctx).GetById()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for character [%d] which does not exist.", e.ActorId)
			return
		}

		// For remaining messenger members.
		go func() {
			for _, m := range p.Members() {
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), messengerLeft(l)(ctx)(wp)(e.Body.Slot))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] has left messenger [%d].", tc.Id(), p.Id())
				}
			}
		}()
		go func() {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.ActorId, messengerLeft(l)(ctx)(wp)(e.Body.Slot))
			if err != nil {
				l.WithError(err).Errorf("Unable to announce character [%d] has left messenger [%d].", tc.Id(), p.Id())
			}
		}()

	}
}

func messengerLeft(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(position byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(position byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(position byte) model.Operator[session.Model] {
			return func(position byte) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationRemoveBody(l)(position))
			}
		}
	}
}

func handleJoin(sc server.Model, wp writer.Producer) message.Handler[messenger2.StatusEvent[messenger2.JoinedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e messenger2.StatusEvent[messenger2.JoinedEventBody]) {
		if e.Type != messenger2.EventMessengerStatusTypeJoined {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := messenger.NewProcessor(l, ctx).GetById(e.MessengerId)
		if err != nil {
			l.WithError(err).Errorf("Received joined event for messenger [%d] which does not exist.", e.MessengerId)
			return
		}
		mm, err := p.FindMember(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Received joined event for messenger [%d] which does not exist.", e.MessengerId)
			return
		}

		cp := character.NewProcessor(l, ctx)
		tc, err := cp.GetById(cp.PetModelDecorator, cp.InventoryDecorator)(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Received joined event for character [%d] which does not exist.", e.ActorId)
			return
		}

		// For remaining messenger members.
		go func() {
			for _, m := range p.Members() {
				if m.Id() == e.ActorId {
					continue
				}
				bp := session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationAddBody(l, ctx)(e.Body.Slot, tc, mm.ChannelId()))
				err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.Id(), bp)
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] has joined messenger [%d].", tc.Id(), p.Id())
				}
			}
		}()
		go func() {
			err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.ActorId, func(s session.Model) error {
				err = session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationJoinBody(l)(e.Body.Slot))(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] has joined messenger [%d].", tc.Id(), p.Id())
				}

				cp := character.NewProcessor(l, ctx)
				for _, m := range p.Members() {
					if m.Id() == e.ActorId {
						continue
					}
					mc, err := cp.GetById(cp.InventoryDecorator, cp.PetModelDecorator)(m.Id())
					if err != nil {
						continue
					}

					err = session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationAddBody(l, ctx)(m.Slot(), mc, m.ChannelId()))(s)
					if err != nil {
						l.WithError(err).Errorf("Unable to announce character [%d] has joined messenger [%d].", tc.Id(), p.Id())
					}
				}
				return nil
			})
			if err != nil {
				l.WithError(err).Errorf("Unable to announce character [%d] has joined messenger [%d].", tc.Id(), p.Id())
			}
		}()
	}
}
