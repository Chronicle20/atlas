package shop

import (
	asset2 "atlas-merchant/kafka/message/asset"
	"atlas-merchant/kafka/message/compartment"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func ReleaseAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, assetId uint32, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.ReleaseCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandRelease,
		Body: compartment.ReleaseCommandBody{
			TransactionId: transactionId,
			AssetId:       assetId,
			Quantity:      quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AcceptAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, templateId uint32, assetData asset2.AssetData) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.AcceptCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandAccept,
		Body: compartment.AcceptCommandBody{
			TransactionId: transactionId,
			TemplateId:    templateId,
			AssetData:     assetData,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
