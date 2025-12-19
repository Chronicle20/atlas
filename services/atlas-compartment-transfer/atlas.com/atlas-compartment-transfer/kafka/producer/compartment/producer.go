package compartment

import (
	"atlas-compartment-transfer/kafka/message/compartment"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// CompletedStatusEventProvider creates a provider for a COMPLETED status event
func CompletedStatusEventProvider(characterId uint32, transactionId uuid.UUID, accountId uint32, assetId uint32, compartmentId uuid.UUID, compartmentType byte, inventoryType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.StatusEvent[compartment.StatusEventCompletedBody]{
		CharacterId: characterId,
		Type:        compartment.StatusEventTypeCompleted,
		Body: compartment.StatusEventCompletedBody{
			TransactionId:   transactionId,
			AccountId:       accountId,
			AssetId:         assetId,
			CompartmentId:   compartmentId,
			CompartmentType: compartmentType,
			InventoryType:   inventoryType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
