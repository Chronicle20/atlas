package account

import (
	"atlas-channel/account"
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("account_status_event")(EnvEventTopicAccountStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				t, _ := topic.EnvProvider(l)(EnvEventTopicAccountStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAccountStatusEvent(sc))))
			}
		}
	}
}

func handleAccountStatusEvent(sc server.Model) func(l logrus.FieldLogger, ctx context.Context, event statusEvent) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent) {
		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if event.Status == EventAccountStatusLoggedIn {
			account.GetRegistry().Login(account.Key{Tenant: t, Id: event.AccountId})
		} else if event.Status == EventAccountStatusLoggedOut {
			account.GetRegistry().Logout(account.Key{Tenant: t, Id: event.AccountId})
		}
	}
}
