package session

import (
	session3 "atlas-login/kafka/message/account/session"
	"atlas-login/kafka/producer"
	session2 "atlas-login/kafka/producer/account/session"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func Create(_ logrus.FieldLogger, kp producer.Provider) func(sessionId uuid.UUID, accountId uint32, accountName string, password string, ipAddress string) error {
	return func(sessionId uuid.UUID, accountId uint32, accountName string, password string, ipAddress string) error {
		return kp(session3.EnvCommandTopic)(session2.CreateCommandProvider(sessionId, accountId, accountName, password, ipAddress))
	}
}

func Destroy(l logrus.FieldLogger, kp producer.Provider) func(sessionId uuid.UUID, accountId uint32) {
	return func(sessionId uuid.UUID, accountId uint32) {
		l.Debugf("Destroying session for account [%d].", accountId)
		_ = kp(session3.EnvCommandTopic)(session2.LogoutCommandProvider(sessionId, accountId))
	}
}

func UpdateState(_ logrus.FieldLogger, kp producer.Provider) func(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) error {
	return func(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) error {
		return kp(session3.EnvCommandTopic)(session2.ProgressStateCommandProvider(sessionId, accountId, state, params))
	}
}
