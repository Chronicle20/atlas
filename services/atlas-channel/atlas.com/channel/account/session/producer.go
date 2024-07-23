package session

import (
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func logoutCommandProvider(tenant tenant.Model, accountId uint32) model.SliceProvider[kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &logoutCommand{
		Tenant:    tenant,
		Issuer:    "channel",
		AccountId: accountId,
	}
	return producer.SingleMessageProvider(key, value)
}
