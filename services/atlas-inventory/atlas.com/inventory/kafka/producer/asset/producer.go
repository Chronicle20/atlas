package asset

import (
	"atlas-inventory/kafka/message/asset"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func CreatedEventStatusProvider(characterId uint32, compartmentId uuid.UUID, assetId uint32, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.CreatedStatusEventBody]{
		CharacterId:   characterId,
		CompartmentId: compartmentId,
		AssetId:       assetId,
		Slot:          slot,
		Type:          asset.StatusEventTypeCreated,
		Body:          asset.CreatedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeletedEventStatusProvider(characterId uint32, compartmentId uuid.UUID, assetId uint32, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.DeletedStatusEventBody]{
		CharacterId:   characterId,
		CompartmentId: compartmentId,
		AssetId:       assetId,
		Slot:          slot,
		Type:          asset.StatusEventTypeDeleted,
		Body:          asset.DeletedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func MovedEventStatusProvider(characterId uint32, compartmentId uuid.UUID, assetId uint32, newSlot int16, oldSlot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.MovedStatusEventBody]{
		CharacterId:   characterId,
		CompartmentId: compartmentId,
		AssetId:       assetId,
		Slot:          newSlot,
		Type:          asset.StatusEventTypeMoved,
		Body: asset.MovedStatusEventBody{
			OldSlot: oldSlot,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func QuantityChangedEventStatusProvider(characterId uint32, compartmentId uuid.UUID, assetId uint32, slot int16, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.QuantityChangedEventBody]{
		CharacterId:   characterId,
		CompartmentId: compartmentId,
		AssetId:       assetId,
		Slot:          slot,
		Type:          asset.StatusEventTypeQuantityChanged,
		Body: asset.QuantityChangedEventBody{
			Quantity: quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
