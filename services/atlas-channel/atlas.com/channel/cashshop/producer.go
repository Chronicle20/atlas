package cashshop

import (
	"atlas-channel/kafka/message/cashshop"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func CharacterEnterCashShopStatusEventProvider(actorId uint32, f field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &cashshop.StatusEvent[cashshop.CharacterMovementBody]{
		WorldId: f.WorldId(),
		Type:    cashshop.EventCashShopStatusTypeCharacterEnter,
		Body: cashshop.CharacterMovementBody{
			CharacterId: actorId,
			ChannelId:   f.ChannelId(),
			MapId:       f.MapId(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CharacterExitCashShopStatusEventProvider(actorId uint32, f field.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(actorId))
	value := &cashshop.StatusEvent[cashshop.CharacterMovementBody]{
		WorldId: f.WorldId(),
		Type:    cashshop.EventCashShopStatusTypeCharacterExit,
		Body: cashshop.CharacterMovementBody{
			CharacterId: actorId,
			ChannelId:   f.ChannelId(),
			MapId:       f.MapId(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RequestPurchaseCommandProvider builds the REQUEST_PURCHASE command.
// transactionId is an opaque correlation id (uuid.Nil for callers with no
// record to correlate back to, e.g. the ordinary BUY arm) — see
// RequestPurchaseCommandBody's doc comment. operation names the requesting
// arm (empty for the ordinary BUY arm and the name-change/world-transfer
// paid legs, which correlate via transactionId instead) — see
// RequestPurchaseCommandBody.Operation's doc comment.
func RequestPurchaseCommandProvider(characterId uint32, serialNumber uint32, currency uint32, transactionId uuid.UUID, operation string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestPurchaseCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestPurchase,
		Body: cashshop.RequestPurchaseCommandBody{
			TransactionId: transactionId,
			Currency:      currency,
			SerialNumber:  serialNumber,
			Operation:     operation,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RequestCouponRedemptionCommandProvider carries the ALREADY-NORMALIZED coupon
// code. The owning account is resolved service-side from CharacterId, and the
// packet's targetCharacter field is deliberately not forwarded — targeted /
// gift redemption is out of scope.
func RequestCouponRedemptionCommandProvider(characterId uint32, code string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestCouponRedemptionCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestCouponRedemption,
		Body: cashshop.RequestCouponRedemptionCommandBody{
			Code: code,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestInventoryIncreaseByTypeCommandProvider(characterId uint32, currency uint32, inventoryType byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestInventoryIncreaseByTypeCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestInventoryIncreaseByType,
		Body: cashshop.RequestInventoryIncreaseByTypeCommandBody{
			Currency:      currency,
			InventoryType: inventoryType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestInventoryIncreaseByItemCommandProvider(characterId uint32, currency uint32, serialNumber uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestInventoryIncreaseByItemCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestInventoryIncreaseByItem,
		Body: cashshop.RequestInventoryIncreaseByItemCommandBody{
			Currency:     currency,
			SerialNumber: serialNumber,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestStorageIncreaseCommandProvider(characterId uint32, currency uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestStorageIncreaseBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestStorageIncrease,
		Body: cashshop.RequestStorageIncreaseBody{
			Currency: currency,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestStorageIncreaseByItemCommandProvider(characterId uint32, currency uint32, serialNumber uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestStorageIncreaseByItemCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestStorageIncreaseByItem,
		Body: cashshop.RequestStorageIncreaseByItemCommandBody{
			Currency:     currency,
			SerialNumber: serialNumber,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestCharacterSlotIncreaseByItemCommandProvider(characterId uint32, currency uint32, serialNumber uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestCharacterSlotIncreaseByItemCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestCharacterSlotIncreaseByItem,
		Body: cashshop.RequestCharacterSlotIncreaseByItemCommandBody{
			Currency:     currency,
			SerialNumber: serialNumber,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func OpenSurpriseCommandProvider(characterId uint32, transactionId uuid.UUID, accountId uint32, cashId int64) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.OpenSurpriseCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeOpenSurprise,
		Body: cashshop.OpenSurpriseCommandBody{
			TransactionId: transactionId,
			AccountId:     accountId,
			CashId:        cashId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestLockerRebateCommandProvider(characterId uint32, transactionId uuid.UUID, accountId uint32, cashId int64) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestLockerRebateCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestLockerRebate,
		Body: cashshop.RequestLockerRebateCommandBody{
			TransactionId: transactionId,
			AccountId:     accountId,
			CashId:        cashId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// AcknowledgeGiftsCommandProvider drains the "gift list presented" flag on
// the named cashIds (task-240 Defect H). Uses characterId only as the
// producer partitioning key, mirroring every other command here -- the
// server-side effect is entirely scoped by accountId + cashIds.
func AcknowledgeGiftsCommandProvider(characterId uint32, accountId uint32, cashIds []int64) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.AcknowledgeGiftsCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeAcknowledgeGifts,
		Body: cashshop.AcknowledgeGiftsCommandBody{
			AccountId: accountId,
			CashIds:   cashIds,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// MarkGiftNoteSentCommandProvider marks cashId's gift-forward note as sent
// (task-240 Defect I). Uses characterId only as the producer partitioning
// key, mirroring every other command here -- the server-side effect is
// entirely scoped by accountId + cashId.
func MarkGiftNoteSentCommandProvider(characterId uint32, accountId uint32, cashId int64) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.MarkGiftNoteSentCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeMarkGiftNoteSent,
		Body: cashshop.MarkGiftNoteSentCommandBody{
			AccountId: accountId,
			CashId:    cashId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestGiftPurchaseCommandProvider(characterId uint32, transactionId uuid.UUID, serialNumber uint32, recipientCharacterId uint32, senderName string, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestGiftPurchaseCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestGiftPurchase,
		Body: cashshop.RequestGiftPurchaseCommandBody{
			TransactionId:        transactionId,
			SerialNumber:         serialNumber,
			RecipientCharacterId: recipientCharacterId,
			SenderName:           senderName,
			Message:              message,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func RequestPackagePurchaseCommandProvider(characterId uint32, transactionId uuid.UUID, currency uint32, serialNumber uint32, recipientCharacterId uint32, senderName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestPackagePurchaseCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestPackagePurchase,
		Body: cashshop.RequestPackagePurchaseCommandBody{
			TransactionId:        transactionId,
			Currency:             currency,
			SerialNumber:         serialNumber,
			RecipientCharacterId: recipientCharacterId,
			SenderName:           senderName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RequestRingPurchaseCommandProvider builds the REQUEST_RING_PURCHASE
// command. transactionId is minted by the caller (once per click, mirroring
// RequestGiftPurchaseCommandProvider/RequestPackagePurchaseCommandProvider's
// idempotency pattern) so a Kafka redelivery replays this id and is
// rejected by atlas-cashshop's ring ledger while a genuine second click
// gets a fresh one. ringType selects "COUPLE" vs "FRIENDSHIP".
func RequestRingPurchaseCommandProvider(characterId uint32, transactionId uuid.UUID, currency uint32, serialNumber uint32, partnerCharacterId uint32, senderName string, message string, ringType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestRingPurchaseCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestRingPurchase,
		Body: cashshop.RequestRingPurchaseCommandBody{
			TransactionId:      transactionId,
			Currency:           currency,
			SerialNumber:       serialNumber,
			PartnerCharacterId: partnerCharacterId,
			SenderName:         senderName,
			Message:            message,
			RingType:           ringType,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// RequestEquipSlotIncreaseCommandProvider builds the
// REQUEST_EQUIP_SLOT_INCREASE command (task-240 task 23, mode 9/10).
func RequestEquipSlotIncreaseCommandProvider(characterId uint32, transactionId uuid.UUID, currency uint32, serialNumber uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestEquipSlotIncreaseCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestEquipSlotIncrease,
		Body: cashshop.RequestEquipSlotIncreaseCommandBody{
			TransactionId: transactionId,
			Currency:      currency,
			SerialNumber:  serialNumber,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
