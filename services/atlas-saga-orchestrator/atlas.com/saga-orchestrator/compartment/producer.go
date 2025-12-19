package compartment

import (
	"atlas-saga-orchestrator/kafka/message/compartment"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"time"
)

func RequestCreateAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.CreateAssetCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandCreateAsset,
		Body: compartment.CreateAssetCommandBody{
			TemplateId:   templateId,
			Quantity:     quantity,
			Expiration:   time.Time{},
			OwnerId:      0,
			Flag:         0,
			Rechargeable: 0,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestDestroyAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.DestroyCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandDestroy,
		Body: compartment.DestroyCommandBody{
			Slot:     slot,
			Quantity: quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestEquipAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.EquipCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandEquip,
		Body: compartment.EquipCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestUnequipAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.UnequipCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandUnequip,
		Body: compartment.UnequipCommandBody{
			Source:      source,
			Destination: destination,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
