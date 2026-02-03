package account

import (
	account2 "atlas-account/kafka/message/account"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"math/rand"
)

func createCommandProvider(name string, password string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(rand.Int())
	value := &account2.Command[account2.CreateCommandBody]{
		Type: account2.CommandTypeCreate,
		Body: account2.CreateCommandBody{
			Name:     name,
			Password: password,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func createdEventProvider() func(accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(account2.EventStatusCreated)
}

func loggedInEventProvider() func(accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(account2.EventStatusLoggedIn)
}

func loggedOutEventProvider() func(accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(account2.EventStatusLoggedOut)
}

func deletedEventProvider() func(accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(account2.EventStatusDeleted)
}

func accountStatusEventProvider(status string) func(accountId uint32, name string) model.Provider[[]kafka.Message] {
	return func(accountId uint32, name string) model.Provider[[]kafka.Message] {
		key := producer.CreateKey(int(accountId))
		value := &account2.StatusEvent{
			AccountId: accountId,
			Name:      name,
			Status:    status,
		}
		return producer.SingleMessageProvider(key, value)
	}
}

func logoutCommandProvider(accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &account2.SessionCommand[account2.LogoutSessionCommandBody]{
		SessionId: uuid.Nil,
		AccountId: accountId,
		Issuer:    account2.SessionCommandIssuerInternal,
		Type:      account2.SessionCommandTypeLogout,
		Body:      account2.LogoutSessionCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func createdStatusProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &account2.SessionStatusEvent[account2.CreatedSessionStatusEventBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Type:      account2.SessionEventStatusTypeCreated,
		Body:      account2.CreatedSessionStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestLicenseAgreementStatusProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &account2.SessionStatusEvent[any]{
		SessionId: sessionId,
		AccountId: accountId,
		Type:      account2.SessionEventStatusTypeRequestLicenseAgreement,
	}
	return producer.SingleMessageProvider(key, value)
}

func stateChangedStatusProvider(sessionId uuid.UUID, accountId uint32, state uint8, params interface{}) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &account2.SessionStatusEvent[account2.StateChangedSessionStatusEventBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Type:      account2.SessionEventStatusTypeStateChanged,
		Body: account2.StateChangedSessionStatusEventBody{
			State:  state,
			Params: params,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func errorStatusProvider(sessionId uuid.UUID, accountId uint32, code string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &account2.SessionStatusEvent[account2.ErrorSessionStatusEventBody]{
		SessionId: sessionId,
		AccountId: accountId,
		Type:      account2.SessionEventStatusTypeError,
		Body: account2.ErrorSessionStatusEventBody{
			Code: code,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
