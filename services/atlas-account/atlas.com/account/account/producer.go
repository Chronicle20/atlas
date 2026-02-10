package account

import (
	account2 "atlas-account/kafka/message/account"
	ban2 "atlas-account/kafka/message/ban"
	"fmt"
	"math/rand"
	"time"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
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

func createdStatusProvider(sessionId uuid.UUID, accountId uint32, accountName string, ipAddress string, hwid string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &account2.SessionStatusEvent[account2.CreatedSessionStatusEventBody]{
		SessionId:   sessionId,
		AccountId:   accountId,
		AccountName: accountName,
		Type:        account2.SessionEventStatusTypeCreated,
		Body: account2.CreatedSessionStatusEventBody{
			IPAddress: ipAddress,
			HWID:      hwid,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestLicenseAgreementStatusProvider(sessionId uuid.UUID, accountId uint32, accountName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &account2.SessionStatusEvent[any]{
		SessionId:   sessionId,
		AccountId:   accountId,
		AccountName: accountName,
		Type:        account2.SessionEventStatusTypeRequestLicenseAgreement,
	}
	return producer.SingleMessageProvider(key, value)
}

func stateChangedStatusProvider(sessionId uuid.UUID, accountId uint32, accountName string, state uint8, params interface{}) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &account2.SessionStatusEvent[account2.StateChangedSessionStatusEventBody]{
		SessionId:   sessionId,
		AccountId:   accountId,
		AccountName: accountName,
		Type:        account2.SessionEventStatusTypeStateChanged,
		Body: account2.StateChangedSessionStatusEventBody{
			State:  state,
			Params: params,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func createBanCommandProvider(accountId uint32, reason string, expiresAt time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &ban2.Command[ban2.CreateCommandBody]{
		Type: ban2.CommandTypeCreate,
		Body: ban2.CreateCommandBody{
			BanType:    2, // Account ban
			Value:      fmt.Sprintf("%d", accountId),
			Reason:     reason,
			ReasonCode: 0,
			Permanent:  false,
			ExpiresAt:  expiresAt,
			IssuedBy:   "atlas-account",
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func banStatusProvider(sessionId uuid.UUID, accountId uint32, accountName string, ipAddress string, hwid string, reason byte, until time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &account2.SessionStatusEvent[account2.ErrorSessionStatusEventBody]{
		SessionId:   sessionId,
		AccountId:   accountId,
		AccountName: accountName,
		Type:        account2.SessionEventStatusTypeError,
		Body: account2.ErrorSessionStatusEventBody{
			Code:      DeletedOrBlocked,
			Reason:    reason,
			Until:     until,
			IPAddress: ipAddress,
			HWID:      hwid,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func errorStatusProvider(sessionId uuid.UUID, accountId uint32, accountName string, code string, ipAddress string, hwid string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &account2.SessionStatusEvent[account2.ErrorSessionStatusEventBody]{
		SessionId:   sessionId,
		AccountId:   accountId,
		AccountName: accountName,
		Type:        account2.SessionEventStatusTypeError,
		Body: account2.ErrorSessionStatusEventBody{
			Code:      code,
			IPAddress: ipAddress,
			HWID:      hwid,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
