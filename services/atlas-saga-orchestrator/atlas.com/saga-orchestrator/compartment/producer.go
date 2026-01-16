package compartment

import (
	"atlas-saga-orchestrator/kafka/message/compartment"
	"atlas-saga-orchestrator/kafka/message/transfer"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"time"
)

func RequestCreateAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.CreateAssetCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandCreateAsset,
		Body: compartment.CreateAssetCommandBody{
			TemplateId:   templateId,
			Quantity:     quantity,
			Expiration:   expiration,
			OwnerId:      0,
			Flag:         0,
			Rechargeable: 0,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestDestroyAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32, removeAll bool) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.DestroyCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventoryType),
		Type:          compartment.CommandDestroy,
		Body: compartment.DestroyCommandBody{
			Slot:      slot,
			Quantity:  quantity,
			RemoveAll: removeAll,
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

func RequestTransferAssetCommandProvider(transactionId uuid.UUID, worldId byte, accountId uint32, characterId uint32, assetId uint32, fromCompartmentId uuid.UUID, fromCompartmentType byte, fromInventoryType string, toCompartmentId uuid.UUID, toCompartmentType byte, toInventoryType string, referenceId uint32, templateId uint32, referenceType string, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &transfer.TransferCommand{
		TransactionId:       transactionId,
		WorldId:             worldId,
		AccountId:           accountId,
		CharacterId:         characterId,
		AssetId:             assetId,
		FromCompartmentId:   fromCompartmentId,
		FromCompartmentType: fromCompartmentType,
		FromInventoryType:   fromInventoryType,
		ToCompartmentId:     toCompartmentId,
		ToCompartmentType:   toCompartmentType,
		ToInventoryType:     toInventoryType,
		ReferenceId:         referenceId,
		TemplateId:          templateId,
		ReferenceType:       referenceType,
		Slot:                slot,
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestAcceptAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, templateId uint32, referenceId uint32, referenceType string, referenceData []byte, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartment.Command[compartment.AcceptCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: inventoryType,
		Type:          compartment.CommandAccept,
		Body: compartment.AcceptCommandBody{
			TransactionId: transactionId,
			TemplateId:    templateId,
			ReferenceId:   referenceId,
			ReferenceType: referenceType,
			ReferenceData: referenceData,
			Quantity:      quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestReleaseAssetCommandProvider(transactionId uuid.UUID, characterId uint32, inventoryType byte, assetId uint32, quantity uint32) model.Provider[[]kafka.Message] {
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
