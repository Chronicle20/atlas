package pickup

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	EnvCommandTopic = "COMMAND_TOPIC_ITEM_CONSUMED_ON_PICKUP"
	CommandType     = "ITEM_CONSUMED_ON_PICKUP"
)

type Command struct {
	TenantId      uuid.UUID `json:"tenantId"`
	CharacterId   uint32    `json:"characterId"`
	ItemId        uint32    `json:"itemId"`
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
}

func NewCommandProvider(tenantId uuid.UUID, characterId uint32, itemId uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &Command{
		TenantId:      tenantId,
		CharacterId:   characterId,
		ItemId:        itemId,
		TransactionId: transactionId,
		Type:          CommandType,
	}
	return producer.SingleMessageProvider(key, value)
}
