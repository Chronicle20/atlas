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

func RequestPurchaseCommandProvider(characterId uint32, serialNumber uint32, currency uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &cashshop.Command[cashshop.RequestPurchaseCommandBody]{
		CharacterId: characterId,
		Type:        cashshop.CommandTypeRequestPurchase,
		Body: cashshop.RequestPurchaseCommandBody{
			Currency:     currency,
			SerialNumber: serialNumber,
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
