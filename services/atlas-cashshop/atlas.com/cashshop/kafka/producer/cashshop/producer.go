package cashshop

import (
	"atlas-cashshop/kafka/message/cashshop"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func ErrorStatusEventProvider(characterId uint32, error string, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.ErrorEventBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeError,
		Body: cashshop.ErrorEventBody{
			Error:         error,
			TransactionId: transactionId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func InventoryCapacityIncreasedStatusEventProvider(characterId uint32, inventoryType byte, capacity uint32, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.InventoryCapacityIncreasedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeInventoryCapacityIncreased,
		Body: cashshop.InventoryCapacityIncreasedBody{
			InventoryType: inventoryType,
			Capacity:      capacity,
			Amount:        amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func PurchaseStatusEventProvider(characterId uint32, templateId, price uint32, compartmentId uuid.UUID, assetId uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.PurchaseEventBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypePurchase,
		Body: cashshop.PurchaseEventBody{
			TemplateId:    templateId,
			Price:         price,
			CompartmentId: compartmentId,
			AssetId:       assetId,
			TransactionId: transactionId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func SurpriseOpenedStatusEventProvider(characterId uint32, compartmentId uuid.UUID, boxCashId int64, boxRemaining uint32, rewardAssetId uint32, rewardTemplateId uint32, rewardCount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.SurpriseOpenedEventBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeSurpriseOpened,
		Body: cashshop.SurpriseOpenedEventBody{
			CompartmentId:    compartmentId,
			BoxCashId:        boxCashId,
			BoxRemaining:     boxRemaining,
			RewardAssetId:    rewardAssetId,
			RewardTemplateId: rewardTemplateId,
			RewardCount:      rewardCount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func SurpriseFailedStatusEventProvider(characterId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.SurpriseFailedEventBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeSurpriseFailed,
		Body: cashshop.SurpriseFailedEventBody{
			Reason: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
