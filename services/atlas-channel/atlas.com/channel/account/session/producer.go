package session

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func logoutCommandProvider(sessionId uuid.UUID, accountId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(accountId))
	value := &logoutCommand{
		SessionId: sessionId,
		Issuer:    "CHANNEL",
		AccountId: accountId,
	}
	return producer.SingleMessageProvider(key, value)
}
