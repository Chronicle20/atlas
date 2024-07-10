package session

import (
	"atlas-account/kafka"
	"atlas-account/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitLogoutCommand(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvCommandTopicAccountLogout))
	return func(accountId uint32) {
		command := &logoutCommand{
			Tenant:    tenant,
			AccountId: accountId,
		}
		p(producer.CreateKey(int(accountId)), command)
	}
}

func emitLoggedInEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, name string) {
	return emitAccountStatusEvent(l, span, tenant)(EventAccountStatusLoggedIn)
}

func emitLoggedOutEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, name string) {
	return emitAccountStatusEvent(l, span, tenant)(EventAccountStatusLoggedOut)
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
