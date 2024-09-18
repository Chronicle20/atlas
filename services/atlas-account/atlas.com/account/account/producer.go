package account

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
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
	return accountStatusEventProvider(EventAccountStatusCreated)
}

func loggedInEventProvider() func(accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(EventAccountStatusLoggedIn)
}

func loggedOutEventProvider() func(accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(EventAccountStatusLoggedOut)
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
