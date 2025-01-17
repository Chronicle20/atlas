package session

import (
	"atlas-channel/kafka/producer"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func Destroy(l logrus.FieldLogger, kp producer.Provider) func(sessionId uuid.UUID, accountId uint32) {
	return func(sessionId uuid.UUID, accountId uint32) {
		l.Debugf("Destroying session for account [%d].", accountId)
		_ = kp(EnvCommandTopic)(logoutCommandProvider(sessionId, accountId))
	}
}

func UpdateState(_ logrus.FieldLogger, kp producer.Provider) func(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) error {
	return func(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) error {
		return kp(EnvCommandTopic)(progressStateCommandProvider(sessionId, accountId, state, params))
	}
}
