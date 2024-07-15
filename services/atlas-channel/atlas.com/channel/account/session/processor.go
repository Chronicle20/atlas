package session

import (
	"atlas-channel/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func Destroy(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32) {
	return func(accountId uint32) {
		l.Debugf("Destroying session for account [%d].", accountId)
		emitLogoutCommand(l, span, tenant)(accountId)
	}
}
