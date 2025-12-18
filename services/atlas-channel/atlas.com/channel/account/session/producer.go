package session

import (
	"atlas-channel/kafka/message/account/session"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func ProgressStateCommandProvider(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &session.Command[session.ProgressStateCommandBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Issuer:    session.CommandIssuerChannel,
		Type:      session.CommandTypeProgressState,
		Body: session.ProgressStateCommandBody{
			State:  state,
			Params: params,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func LogoutCommandProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &session.Command[session.LogoutCommandBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Issuer:    session.CommandIssuerChannel,
		Type:      session.CommandTypeLogout,
		Body:      session.LogoutCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
