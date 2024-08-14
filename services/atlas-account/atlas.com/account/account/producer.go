package account

import (
	"atlas-account/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"math/rand"
)

func createCommandProvider(tenant tenant.Model, name string, password string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(rand.Int())
	value := &createCommand{
		Tenant:   tenant,
		Name:     name,
		Password: password,
	}
	return producer.SingleMessageProvider(key, value)
}

func createdEventProvider() func(tenant tenant.Model, accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(EventAccountStatusCreated)
}

func loggedInEventProvider() func(tenant tenant.Model, accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(EventAccountStatusLoggedIn)
}

func loggedOutEventProvider() func(tenant tenant.Model, accountId uint32, name string) model.Provider[[]kafka.Message] {
	return accountStatusEventProvider(EventAccountStatusLoggedOut)
}

func accountStatusEventProvider(status string) func(tenant tenant.Model, accountId uint32, name string) model.Provider[[]kafka.Message] {
	return func(tenant tenant.Model, accountId uint32, name string) model.Provider[[]kafka.Message] {
		key := producer.CreateKey(int(accountId))
		value := &statusEvent{
			Tenant:    tenant,
			AccountId: accountId,
			Name:      name,
			Status:    status,
		}
		return producer.SingleMessageProvider(key, value)
	}
}
