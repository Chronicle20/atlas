package session

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func logoutCommandProvider(accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &command[logoutCommandBody]{
		SessionId: uuid.Nil,
		AccountId: accountId,
		Issuer:    CommandIssuerInternal,
		Type:      CommandTypeLogout,
		Body:      logoutCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func createdStatusProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &statusEvent[createdStatusEventBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Type:      EventStatusTypeCreated,
		Body:      createdStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestLicenseAgreementStatusProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &statusEvent[any]{
		SessionId: sessionId,
		AccountId: accountId,
		Type:      EventStatusTypeRequestLicenseAgreement,
	}
	return producer.SingleMessageProvider(key, value)
}

func errorStatusProvider(sessionId uuid.UUID, accountId uint32, code string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &statusEvent[errorStatusEventBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Type:      EventStatusTypeError,
		Body: errorStatusEventBody{
			Code: code,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
