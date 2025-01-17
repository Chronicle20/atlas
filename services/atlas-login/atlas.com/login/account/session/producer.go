package session

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func createCommandProvider(sessionId uuid.UUID, accountId uint32, accountName string, password string, ipAddress string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &command[createCommandBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Issuer:    CommandIssuerLogin,
		Type:      CommandTypeCreate,
		Body: createCommandBody{
			AccountName: accountName,
			Password:    password,
			IPAddress:   ipAddress,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func logoutCommandProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &command[logoutCommandBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Issuer:    CommandIssuerLogin,
		Type:      CommandTypeLogout,
		Body:      logoutCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
