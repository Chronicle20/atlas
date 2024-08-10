package account

import (
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/server"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/opentracing/opentracing-go"
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

func AccountStatusRegister(l logrus.FieldLogger, sc server.Model) (string, handler.Handler) {
	t, _ := topic.EnvProvider(l)(EnvEventTopicAccountStatus)()
	return t, message.AdaptHandler(message.PersistentConfig(handleAccountStatusEvent(sc)))
}

func handleAccountStatusEvent(sc server.Model) func(l logrus.FieldLogger, span opentracing.Span, event statusEvent) {
	return func(l logrus.FieldLogger, span opentracing.Span, event statusEvent) {
		if sc.Tenant().Id != event.Tenant.Id {
			return
		}
		if sc.Tenant().Region != event.Tenant.Region {
			return
		}
		if sc.Tenant().MajorVersion != event.Tenant.MajorVersion {
			return
		}
		if sc.Tenant().MinorVersion != event.Tenant.MinorVersion {
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
