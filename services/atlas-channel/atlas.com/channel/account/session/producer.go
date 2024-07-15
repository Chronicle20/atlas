package session

import (
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitLogoutCommand(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32) {
	p := producer.ProduceEvent(l, span, lookupTopic(l)(EnvCommandTopicAccountLogout))
	return func(accountId uint32) {
		command := &logoutCommand{
			Tenant:    tenant,
			AccountId: accountId,
		}
		p(producer.CreateKey(int(accountId)), command)
	}
}
