package account

import (
	"atlas-login/account"
	consumer2 "atlas-login/kafka/consumer"
	account2 "atlas-login/kafka/message/account"
	"atlas-login/listener"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("account_status_event")(account2.EnvEventTopicAccountStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(ten tenant.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(ten tenant.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var handles []listener.HandlerHandle
				t, _ := topic.EnvProvider(l)(account2.EnvEventTopicAccountStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleAccountStatusEvent(ten))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleAccountStatusEvent(t tenant.Model) func(l logrus.FieldLogger, ctx context.Context, event account2.StatusEvent) {
	return func(l logrus.FieldLogger, ctx context.Context, event account2.StatusEvent) {
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		if event.Status == account2.EventAccountStatusLoggedIn {
			account.GetRegistry().Login(account.Key{Tenant: t, Id: event.AccountId})
		} else if event.Status == account2.EventAccountStatusLoggedOut {
			account.GetRegistry().Logout(account.Key{Tenant: t, Id: event.AccountId})
		}
	}
}
