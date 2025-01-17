package session

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func progressStateCommandProvider(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &command[progressStateCommandBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Issuer:    CommandIssuerChannel,
		Type:      CommandTypeProgressState,
		Body: progressStateCommandBody{
			State:  state,
			Params: params,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func logoutCommandProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &command[logoutCommandBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Issuer:    CommandIssuerChannel,
		Type:      CommandTypeLogout,
		Body:      logoutCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
