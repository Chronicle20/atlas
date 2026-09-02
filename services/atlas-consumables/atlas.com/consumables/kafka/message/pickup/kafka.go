package pickup

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
)

const (
	EnvCommandTopic topic.Token = "COMMAND_TOPIC_ITEM_CONSUMED_ON_PICKUP"
)

const (
	CommandType = "ITEM_CONSUMED_ON_PICKUP"
)

type Command struct {
	TenantId      uuid.UUID `json:"tenantId"`
	CharacterId   uint32    `json:"characterId"`
	ItemId        uint32    `json:"itemId"`
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
}
