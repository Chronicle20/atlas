package account

import (
	"atlas-account/kafka"
	"atlas-account/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"math/rand"
)

func emitCreateCommand(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string, password string) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvCommandTopicCreateAccount))
	return func(name string, password string) {
		command := &createCommand{
			Tenant:   tenant,
			Name:     name,
			Password: password,
		}
		p(producer.CreateKey(rand.Int()), command)
	}
}

func emitCreatedEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, name string) {
	return emitAccountStatusEvent(l, span, tenant)(EventAccountStatusCreated)
}

func emitAccountStatusEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(status string) func(accountId uint32, name string) {
	return func(status string) func(accountId uint32, name string) {
		p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicAccountStatus))
		return func(accountId uint32, name string) {
			event := &statusEvent{
				Tenant:    tenant,
				AccountId: accountId,
				Name:      name,
				Status:    status,
			}
			p(producer.CreateKey(int(accountId)), event)
		}
	}
}
