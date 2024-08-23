package account

import (
	consumer2 "atlas-login/kafka/consumer"
	"atlas-login/tenant"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/sirupsen/logrus"
)

const (
	consumerNameAccountStatus = "account-status"
)

func AccountStatusConsumer(l logrus.FieldLogger) func(groupId string) consumer.Config {
	return func(groupId string) consumer.Config {
		return consumer2.NewConfig(l)(consumerNameAccountStatus)(EnvEventTopicAccountStatus)(groupId)
	}
}

func AccountStatusRegister(l logrus.FieldLogger, tenant tenant.Model) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicAccountStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleAccountStatusEvent(tenant)))
}

func handleAccountStatusEvent(tenant tenant.Model) func(l logrus.FieldLogger, ctx context.Context, event statusEvent) {
	return func(l logrus.FieldLogger, ctx context.Context, event statusEvent) {
		if tenant.Id != event.Tenant.Id {
			return
		}
		if tenant.Region != event.Tenant.Region {
			return
		}
		if tenant.MajorVersion != event.Tenant.MajorVersion {
			return
		}
		if tenant.MinorVersion != event.Tenant.MinorVersion {
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
}
