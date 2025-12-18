package character

import (
	consumer2 "atlas-buddies/kafka/consumer"
	"atlas-buddies/kafka/message/character"
	"atlas-buddies/list"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(character.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(character.EnvEventTopicStatus)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged(db))))
		}
	}
}

func handleStatusEventCreated(db *gorm.DB) message.Handler[character.StatusEvent[character.CreatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character.StatusEvent[character.CreatedStatusEventBody]) {
		if e.Type != character.StatusEventTypeCreated {
			return
		}
		_, _ = list.NewProcessor(l, ctx, db).Create(e.CharacterId, 30)
	}
}

func handleStatusEventDeleted(db *gorm.DB) message.Handler[character.StatusEvent[character.DeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character.StatusEvent[character.DeletedStatusEventBody]) {
		if e.Type != character.StatusEventTypeDeleted {
			return
		}

		err := list.NewProcessor(l, ctx, db).DeleteAndEmit(e.CharacterId, e.WorldId)
		if err != nil {
			l.WithError(err).Errorf("Unable to delete for character [%d].", e.CharacterId)
		}
	}
}

func handleStatusEventLogin(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event character.StatusEvent[character.LoginStatusEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event character.StatusEvent[character.LoginStatusEventBody]) {
		if event.Type != character.StatusEventTypeLogin {
			return
		}
		err := list.NewProcessor(l, ctx, db).UpdateBuddyChannelAndEmit(event.CharacterId, event.WorldId, int8(event.Body.ChannelId))
		if err != nil {
			l.WithError(err).Errorf("Unable to process login for character [%d].", event.CharacterId)
		}
	}
}

func handleStatusEventLogout(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event character.StatusEvent[character.LogoutStatusEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event character.StatusEvent[character.LogoutStatusEventBody]) {
		if event.Type != character.StatusEventTypeLogout {
			return
		}
		err := list.NewProcessor(l, ctx, db).UpdateBuddyChannelAndEmit(event.CharacterId, event.WorldId, -1)
		if err != nil {
			l.WithError(err).Errorf("Unable to process logout for character [%d].", event.CharacterId)
		}
	}
}

func handleStatusEventChannelChanged(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, event character.StatusEvent[character.ChannelChangedStatusEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, event character.StatusEvent[character.ChannelChangedStatusEventBody]) {
		if event.Type != character.StatusEventTypeChannelChanged {
			return
		}
		if event.Body.ChannelId == event.Body.OldChannelId {
			return
		}
		err := list.NewProcessor(l, ctx, db).UpdateBuddyChannelAndEmit(event.CharacterId, event.WorldId, int8(event.Body.ChannelId))
		if err != nil {
			l.WithError(err).Errorf("Unable to process change channel for character [%d].", event.CharacterId)
		}
	}
}
