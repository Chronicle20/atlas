package session

import (
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func logoutCommandProvider(tenant tenant.Model, sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &logoutCommand{
		Tenant:    tenant,
		SessionId: sessionId,
		Issuer:    "LOGIN",
		AccountId: accountId,
	}
	return producer.SingleMessageProvider(key, value)
}
