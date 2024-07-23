package session

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/tenant"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func Destroy(l logrus.FieldLogger, kp producer.Provider) func(tenant tenant.Model, accountId uint32) {
	return func(tenant tenant.Model, accountId uint32) {
		l.Debugf("Destroying session for account [%d].", accountId)
		_ = kp(EnvCommandTopicAccountLogout)(logoutCommandProvider(tenant, accountId))
	}
}

func UpdateState(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(sessionId uuid.UUID, accountId uint32, state int) (Model, error) {
	return func(sessionId uuid.UUID, accountId uint32, state int) (Model, error) {
		return updateState(l, span, tenant)(sessionId, accountId, state)
	}
}
