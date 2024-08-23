package session

import (
	"atlas-channel/kafka/producer"
	"atlas-channel/tenant"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func Destroy(l logrus.FieldLogger, kp producer.Provider) func(tenant tenant.Model, sessionId uuid.UUID, accountId uint32) {
	return func(tenant tenant.Model, sessionId uuid.UUID, accountId uint32) {
		l.Debugf("Destroying session for account [%d].", accountId)
		_ = kp(EnvCommandTopicAccountLogout)(logoutCommandProvider(tenant, sessionId, accountId))
	}
}

func UpdateState(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(sessionId uuid.UUID, accountId uint32, state int) (Model, error) {
	return func(sessionId uuid.UUID, accountId uint32, state int) (Model, error) {
		return updateState(l, ctx, tenant)(sessionId, accountId, state)
	}
}
