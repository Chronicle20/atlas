package compartment

import (
	"atlas-compartment-transfer/kafka/message/character/compartment"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func AcceptCommandProvider(characterId uint32, compartmentType byte, transactionId uuid.UUID, referenceId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.AcceptCommandBody]{
		CharacterId:   characterId,
		InventoryType: compartmentType,
		Type:          compartment.CommandAccept,
		Body: compartment.AcceptCommandBody{
			TransactionId: transactionId,
			ReferenceId:   referenceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ReleaseCommandProvider(characterId uint32, compartmentType byte, transactionId uuid.UUID, referenceId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.ReleaseCommandBody]{
		CharacterId:   characterId,
		InventoryType: compartmentType,
		Type:          compartment.CommandRelease,
		Body: compartment.ReleaseCommandBody{
			TransactionId: transactionId,
			AssetId:       referenceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
