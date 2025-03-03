package messenger

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
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
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("messenger_status_event")(EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvEventStatusTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleLeft(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleJoin(sc, wp))))
				//_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleError(sc, wp))))
			}
		}
	}
}

func handleLeft(sc server.Model, wp writer.Producer) message.Handler[statusEvent[leftEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[leftEventBody]) {
		if e.Type != EventMessengerStatusTypeLeft {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := messenger.GetById(l)(ctx)(e.MessengerId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for messenger [%d] which does not exist.", e.MessengerId)
			return
		}

		tc, err := character.GetById(l)(ctx)()(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Received left event for character [%d] which does not exist.", e.ActorId)
			return
		}

		// For remaining messenger members.
		go func() {
			for _, m := range p.Members() {
				err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(m.Id(), messengerLeft(l)(ctx)(wp)(e.Body.Slot))
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] has left messenger [%d].", tc.Id(), p.Id())
				}
			}
		}()
		go func() {
			err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.ActorId, messengerLeft(l)(ctx)(wp)(e.Body.Slot))
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
				return session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationRemoveBody(position))
			}
		}
	}
}

func handleJoin(sc server.Model, wp writer.Producer) message.Handler[statusEvent[joinedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[joinedEventBody]) {
		if e.Type != EventMessengerStatusTypeJoined {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
			return
		}

		p, err := messenger.GetById(l)(ctx)(e.MessengerId)
		if err != nil {
			l.WithError(err).Errorf("Received joined event for messenger [%d] which does not exist.", e.MessengerId)
			return
		}
		mm, err := p.FindMember(e.ActorId)
		if err != nil {
			l.WithError(err).Errorf("Received joined event for messenger [%d] which does not exist.", e.MessengerId)
			return
		}

		tc, err := character.GetByIdWithInventory(l)(ctx)()(e.ActorId)
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
				bp := session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationAddBody(ctx)(e.Body.Slot, tc, byte(mm.ChannelId())))
				err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(m.Id(), bp)
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] has joined messenger [%d].", tc.Id(), p.Id())
				}
			}
		}()
		go func() {
			err = session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.ActorId, func(s session.Model) error {
				err = session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationJoinBody(e.Body.Slot))(s)
				if err != nil {
					l.WithError(err).Errorf("Unable to announce character [%d] has joined messenger [%d].", tc.Id(), p.Id())
				}

				for _, m := range p.Members() {
					if m.Id() == e.ActorId {
						continue
					}
					mc, err := character.GetByIdWithInventory(l)(ctx)()(m.Id())
					if err != nil {
						continue
					}

					err = session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationAddBody(ctx)(m.Slot(), mc, byte(m.ChannelId())))(s)
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

//func handleError(sc server.Model, wp writer.Producer) message.Handler[statusEvent[errorEventBody]] {
//	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[errorEventBody]) {
//		if e.Type != EventMessengerStatusTypeError {
//			return
//		}
//
//		if !sc.IsWorld(tenant.MustFromContext(ctx), world.Id(e.WorldId)) {
//			return
//		}
//
//		session.IfPresentByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.ActorId, messengerError(l)(ctx)(wp)(e.Body.Type, e.Body.CharacterName))
//	}
//}
//
//func messengerError(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(errorType string, name string) model.Operator[session.Model] {
//	return func(ctx context.Context) func(wp writer.Producer) func(errorType string, name string) model.Operator[session.Model] {
//		return func(wp writer.Producer) func(errorType string, name string) model.Operator[session.Model] {
//			return func(errorType string, name string) model.Operator[session.Model] {
//				return session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerErrorBody(l)(errorType, name))
//			}
//		}
//	}
//}
