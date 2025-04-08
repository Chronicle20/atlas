package invite

import (
	consumer2 "atlas-buddies/kafka/consumer"
	invite2 "atlas-buddies/kafka/message/invite"
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
			rf(consumer2.NewConfig(l)("invite_status_event")(invite2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(rf func(topic string, handler handler.Handler) (string, error)) {
			var t string
			t, _ = topic.EnvProvider(l)(invite2.EnvEventStatusTopic)()
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAcceptedStatusEvent(db))))
			_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRejectedStatusEvent(db))))
		}
	}
}

func handleAcceptedStatusEvent(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, e invite2.StatusEvent[invite2.AcceptedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e invite2.StatusEvent[invite2.AcceptedEventBody]) {
		if e.Type != invite2.EventInviteStatusTypeAccepted {
			return
		}

		if e.InviteType != invite2.InviteTypeBuddy {
			return
		}

		_ = list.AcceptInvite(l)(ctx)(db)(e.Body.TargetId, e.WorldId, e.Body.OriginatorId)
	}
}

func handleRejectedStatusEvent(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, e invite2.StatusEvent[invite2.RejectedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e invite2.StatusEvent[invite2.RejectedEventBody]) {
		if e.Type != invite2.EventInviteStatusTypeRejected {
			return
		}

		if e.InviteType != invite2.InviteTypeBuddy {
			return
		}

		_ = list.DeleteBuddy(l)(ctx)(db)(e.Body.OriginatorId, e.WorldId, e.Body.TargetId)
	}
}
