package account

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"math/rand"
)

func createCommandProvider(name string, password string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(rand.Int())
	value := &createCommand{
		Name:     name,
		Password: password,
	}
	return producer.SingleMessageProvider(key, value)
}

func createdEventProvider() func(accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(EventStatusCreated)
}

func loggedInEventProvider() func(accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(EventStatusLoggedIn)
}

func loggedOutEventProvider() func(accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(EventStatusLoggedOut)
}

func accountStatusEventProvider(status string) func(accountId uint32, name string) model.Provider[[]kafka.Message] {
	return func(accountId uint32, name string) model.Provider[[]kafka.Message] {
		key := producer.CreateKey(int(accountId))
		value := &statusEvent{
			AccountId: accountId,
			Name:      name,
			Status:    status,
		}
		return producer.SingleMessageProvider(key, value)
	}
}

func logoutCommandProvider(accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &accountSessionCommand[logoutCommandBody]{
		SessionId: uuid.Nil,
		AccountId: accountId,
		Issuer:    SessionCommandIssuerInternal,
		Type:      SessionCommandTypeLogout,
		Body:      logoutCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func createdStatusProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &sessionStatusEvent[createdSessionStatusEventBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Type:      SessionEventStatusTypeCreated,
		Body:      createdSessionStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestLicenseAgreementStatusProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &sessionStatusEvent[any]{
		SessionId: sessionId,
		AccountId: accountId,
		Type:      SessionEventStatusTypeRequestLicenseAgreement,
	}
	return producer.SingleMessageProvider(key, value)
}

func stateChangedStatusProvider(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &sessionStatusEvent[stateChangedSessionStatusEventBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Type:      SessionEventStatusTypeStateChanged,
		Body: stateChangedSessionStatusEventBody{
			State:  state,
			Params: params,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func errorStatusProvider(sessionId uuid.UUID, accountId uint32, code string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &sessionStatusEvent[errorSessionStatusEventBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Type:      SessionEventStatusTypeError,
		Body: errorSessionStatusEventBody{
			Code: code,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
