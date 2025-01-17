package buddylist

import (
	"atlas-channel/buddylist"
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/server"
	"atlas-channel/session"
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

const consumerStatusEvent = "buddy_list_status"

func StatusEventConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerStatusEvent)(EnvStatusEventTopic)(groupId)
	}
}

func StatusEventBuddyAddedRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventBuddyAdded(sc, wp)))
	}
}

func handleStatusEventBuddyAdded(sc server.Model, wp writer.Producer) message.Handler[statusEvent[buddyAddedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c statusEvent[buddyAddedStatusEventBody]) {
		if c.Type != StatusEventTypeBuddyAdded {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != c.WorldId {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(c.CharacterId, redrawBuddyList(l)(ctx)(wp)())
	}
}

func redrawBuddyList(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func() model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func() model.Operator[session.Model] {
		t := tenant.MustFromContext(ctx)
		return func(wp writer.Producer) func() model.Operator[session.Model] {
			buddyOperationFunc := session.Announce(l)(ctx)(wp)(writer.BuddyOperation)
			return func() model.Operator[session.Model] {
				return func(s session.Model) error {
					bl, err := buddylist.GetById(l)(ctx)(s.CharacterId())
					if err != nil {
						return err
					}

					err = buddyOperationFunc(s, writer.BuddyUpdateBody(l, t)(bl.Buddies()))
					if err != nil {
						l.WithError(err).Errorf("Unable to write character [%d] buddy list.", s.CharacterId())
						return err
					}
					return nil
				}
			}
		}
	}
}

func StatusEventBuddyRemovedRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventBuddyRemoved(sc, wp)))
	}
}

func handleStatusEventBuddyRemoved(sc server.Model, wp writer.Producer) message.Handler[statusEvent[buddyRemovedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c statusEvent[buddyRemovedStatusEventBody]) {
		if c.Type != StatusEventTypeBuddyRemoved {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != c.WorldId {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(c.CharacterId, redrawBuddyList(l)(ctx)(wp)())
	}
}

func StatusEventBuddyChannelChangeRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventBuddyChannelChange(sc, wp)))
	}
}

func handleStatusEventBuddyChannelChange(sc server.Model, wp writer.Producer) message.Handler[statusEvent[buddyChannelChangeStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c statusEvent[buddyChannelChangeStatusEventBody]) {
		if c.Type != StatusEventTypeBuddyChannelChange {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != c.WorldId {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(c.CharacterId, buddyChannelChange(l)(ctx)(wp)(c.Body.CharacterId, c.Body.ChannelId))
	}
}

func buddyChannelChange(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, channelId int8) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(characterId uint32, channelId int8) model.Operator[session.Model] {
		return func(wp writer.Producer) func(characterId uint32, channelId int8) model.Operator[session.Model] {
			buddyOperationFunc := session.Announce(l)(ctx)(wp)(writer.BuddyOperation)
			return func(characterId uint32, channelId int8) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := buddyOperationFunc(s, writer.BuddyChannelChangeBody(l)(characterId, channelId))
					if err != nil {
						l.WithError(err).Errorf("Unable to announce character [%d] buddy [%d] channel change to [%d].", s.CharacterId(), characterId, channelId)
						return err
					}
					return nil
				}
			}
		}
	}
}

func StatusEventBuddyCapacityChangeRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventBuddyCapacityChange(sc, wp)))
	}
}

func handleStatusEventBuddyCapacityChange(sc server.Model, wp writer.Producer) message.Handler[statusEvent[buddyCapacityChangeStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c statusEvent[buddyCapacityChangeStatusEventBody]) {
		if c.Type != StatusEventTypeBuddyCapacityUpdate {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != c.WorldId {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(c.CharacterId, buddyCapacityChange(l)(ctx)(wp)(c.Body.Capacity))
	}
}

func buddyCapacityChange(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(capacity byte) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(capacity byte) model.Operator[session.Model] {
		return func(wp writer.Producer) func(capacity byte) model.Operator[session.Model] {
			buddyOperationFunc := session.Announce(l)(ctx)(wp)(writer.BuddyOperation)
			return func(capacity byte) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := buddyOperationFunc(s, writer.BuddyCapacityUpdateBody(l)(capacity))
					if err != nil {
						l.WithError(err).Errorf("Unable to announce character [%d] buddy list capacity [%s] update.", s.CharacterId(), capacity)
						return err
					}
					return nil
				}
			}
		}
	}
}

func StatusEventBuddyErrorRegister(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger) (string, handler.Handler) {
	return func(l logrus.FieldLogger) (string, handler.Handler) {
		t, _ := topic.EnvProvider(l)(EnvStatusEventTopic)()
		return t, message.AdaptHandler(message.PersistentConfig(handleStatusEventBuddyError(sc, wp)))
	}
}

func handleStatusEventBuddyError(sc server.Model, wp writer.Producer) message.Handler[statusEvent[errorStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c statusEvent[errorStatusEventBody]) {
		if c.Type != StatusEventTypeError {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if sc.WorldId() != c.WorldId {
			return
		}

		session.IfPresentByCharacterId(t, sc.WorldId(), sc.ChannelId())(c.CharacterId, buddyError(l)(ctx)(wp)(c.Body.Error))
	}
}

func buddyError(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(errorCode string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(errorCode string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(errorCode string) model.Operator[session.Model] {
			buddyOperationFunc := session.Announce(l)(ctx)(wp)(writer.BuddyOperation)
			return func(errorCode string) model.Operator[session.Model] {
				return func(s session.Model) error {
					err := buddyOperationFunc(s, writer.BuddyErrorBody(l)(errorCode))
					if err != nil {
						l.WithError(err).Errorf("Unable to announce character [%d] error [%s].", s.CharacterId(), errorCode)
						return err
					}
					return nil
				}
			}
		}
	}
}
