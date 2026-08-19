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

// ErrorStatusEventForOperationProvider is ErrorStatusEventProvider with the
// failing arm attached (ErrorEventBody.Operation), so the channel can answer
// on that arm's own *_FAILED mode byte instead of the legacy
// capacity-increase fallback. ErrorStatusEventProvider is left untouched --
// its callers predate this field and keep the empty-operation behavior.
func ErrorStatusEventForOperationProvider(characterId uint32, operation string, error string, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.ErrorEventBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeError,
		Body: cashshop.ErrorEventBody{
			Error:         error,
			TransactionId: transactionId,
			Operation:     operation,
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

// PurchaseStatusEventProvider builds the PURCHASE status event. operation
// echoes RequestPurchaseCommandBody.Operation (empty for the generic BUY
// arm) so the channel can answer on that arm's own SUCCESS mode byte instead
// of the generic purchase-success fallback.
func PurchaseStatusEventProvider(characterId uint32, templateId, price uint32, compartmentId uuid.UUID, assetId uint32, transactionId uuid.UUID, operation string) model.Provider[[]kafka.Message] {
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
			Operation:     operation,
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

// LockerRebatedStatusEventProvider builds the LOCKER_REBATED status event.
// amount is the commodity price refunded; currency is the wallet bucket it
// landed on (see LockerRebatedBody's doc comment).
func LockerRebatedStatusEventProvider(characterId uint32, cashId int64, amount int32, currency uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.LockerRebatedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeLockerRebated,
		Body: cashshop.LockerRebatedBody{
			TransactionId: transactionId,
			CashId:        cashId,
			Amount:        amount,
			Currency:      currency,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// GiftPurchasedStatusEventProvider builds the GIFT_PURCHASED status event.
// characterId is the SENDER (the actor keying the outbound message, same
// convention as every other status event here); recipientCharacterId and
// recipientName identify who received the item.
func GiftPurchasedStatusEventProvider(characterId uint32, transactionId uuid.UUID, recipientName string, templateId uint32, quantity uint16, price uint32, recipientCharacterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.GiftPurchasedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeGiftPurchased,
		Body: cashshop.GiftPurchasedBody{
			TransactionId:        transactionId,
			RecipientName:        recipientName,
			TemplateId:           templateId,
			Quantity:             quantity,
			Price:                price,
			RecipientCharacterId: recipientCharacterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// PackagePurchasedStatusEventProvider builds the PACKAGE_PURCHASED status
// event (task-240 task 16). characterId is the BUYER (the actor keying the
// outbound message, same convention as every other status event here);
// recipientCharacterId/recipientName identify who received the members --
// the buyer's own identity on a buy-for-self purchase, mirroring
// PackagePurchasedBody's doc comment.
func PackagePurchasedStatusEventProvider(characterId uint32, transactionId uuid.UUID, compartmentId uuid.UUID, assetIds []uint32, packageTemplateId uint32, price uint32, recipientCharacterId uint32, recipientName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.PackagePurchasedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypePackagePurchased,
		Body: cashshop.PackagePurchasedBody{
			TransactionId:        transactionId,
			CompartmentId:        compartmentId,
			AssetIds:             assetIds,
			PackageTemplateId:    packageTemplateId,
			Price:                price,
			RecipientCharacterId: recipientCharacterId,
			RecipientName:        recipientName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RingPurchasedStatusEventProvider builds the RING_PURCHASED status event
// (task-240 task 19). characterId is the BUYER (the actor keying the
// outbound message, same convention as every other status event here);
// compartmentId/assetId name the asset created in the BUYER's own locker.
func RingPurchasedStatusEventProvider(characterId uint32, transactionId uuid.UUID, compartmentId uuid.UUID, assetId uint32, partnerName string, templateId uint32, quantity uint16, ringType string, pairId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.RingPurchasedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeRingPurchased,
		Body: cashshop.RingPurchasedBody{
			TransactionId: transactionId,
			CompartmentId: compartmentId,
			AssetId:       assetId,
			PartnerName:   partnerName,
			TemplateId:    templateId,
			Quantity:      quantity,
			RingType:      ringType,
			PairId:        pairId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// EquipSlotIncreasedStatusEventProvider builds the EQUIP_SLOT_INCREASED
// status event (task-240 task 23). slotIndex is the Atlas canonical
// equipped-inventory position (R1, the pendant2 constant) -- NOT the wire
// value the channel must send (always 0).
func EquipSlotIncreasedStatusEventProvider(characterId uint32, transactionId uuid.UUID, slotIndex int16, days uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.StatusEvent[cashshop.EquipSlotIncreasedBody]{
		CharacterId: characterId,
		Type:        cashshop.StatusEventTypeEquipSlotIncreased,
		Body: cashshop.EquipSlotIncreasedBody{
			TransactionId: transactionId,
			SlotIndex:     slotIndex,
			Days:          days,
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
