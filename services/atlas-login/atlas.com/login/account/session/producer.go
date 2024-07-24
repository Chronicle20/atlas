package session

import (
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func logoutCommandProvider(tenant tenant.Model, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &logoutCommand{
		Tenant:    tenant,
		Issuer:    "login",
		AccountId: accountId,
	}
	return producer.SingleMessageProvider(key, value)
}
