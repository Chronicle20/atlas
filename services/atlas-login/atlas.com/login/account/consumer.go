package account

import (
	consumer2 "atlas-login/kafka/consumer"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	consumerNameAccountStatus = "account-status"
)

func StatusConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerNameAccountStatus)(EnvEventTopicAccountStatus)(groupId)
	}
}

func StatusRegister(l logrus.FieldLogger) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicAccountStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleAccountStatusEvent))
}

func handleAccountStatusEvent(l logrus.FieldLogger, ctx context.Context, event statusEvent) {
	t, err := tenant.FromContext(ctx)()
	if err != nil {
		l.WithError(err).Error("error getting tenant")
		return
	}

	if !t.Is(event.Tenant) {
		return
	}

	if event.Status == EventAccountStatusLoggedIn {
		getRegistry().Login(Key{
			Tenant: event.Tenant,
			Id:     event.AccountId,
		})
	} else if event.Status == EventAccountStatusLoggedOut {
		getRegistry().Logout(Key{
			Tenant: event.Tenant,
			Id:     event.AccountId,
		})
	}
}
