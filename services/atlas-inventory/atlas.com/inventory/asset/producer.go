package asset

import (
	"atlas-inventory/kafka/message/asset"
	"time"

	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func makeAssetData(a Model) asset.AssetData {
	return asset.AssetData{
		Expiration:     a.expiration,
		CreatedAt:      a.createdAt,
		Quantity:       a.quantity,
		OwnerId:        a.ownerId,
		Flag:           a.flag,
		Rechargeable:   a.rechargeable,
		Strength:       a.strength,
		Dexterity:      a.dexterity,
		Intelligence:   a.intelligence,
		Luck:           a.luck,
		Hp:             a.hp,
		Mp:             a.mp,
		WeaponAttack:   a.weaponAttack,
		MagicAttack:    a.magicAttack,
		WeaponDefense:  a.weaponDefense,
		MagicDefense:   a.magicDefense,
		Accuracy:       a.accuracy,
		Avoidability:   a.avoidability,
		Hands:          a.hands,
		Speed:          a.speed,
		Jump:           a.jump,
		Slots:     a.slots,
		LevelType: a.levelType,
		Level:          a.level,
		Experience:     a.experience,
		HammersApplied: a.hammersApplied,
		EquippedSince:  a.equippedSince,
		CashId:         a.cashId,
		CommodityId:    a.commodityId,
		PurchaseBy:     a.purchaseBy,
		PetId:          a.petId,
	}
}

func CreatedEventStatusProvider(transactionId uuid.UUID, characterId uint32, a Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(a.Id()))
	value := &asset.StatusEvent[asset.CreatedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: a.CompartmentId(),
		AssetId:       a.Id(),
		TemplateId:    a.TemplateId(),
		Slot:          a.Slot(),
		Type:          asset.StatusEventTypeCreated,
		Body: asset.CreatedStatusEventBody{
			AssetData: makeAssetData(a),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeletedEventStatusProvider(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, assetId uint32, templateId uint32, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.DeletedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: compartmentId,
		AssetId:       assetId,
		TemplateId:    templateId,
		Slot:          slot,
		Type:          asset.StatusEventTypeDeleted,
		Body:          asset.DeletedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func MovedEventStatusProvider(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, assetId uint32, templateId uint32, newSlot int16, oldSlot int16, createdAt time.Time) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.MovedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: compartmentId,
		AssetId:       assetId,
		TemplateId:    templateId,
		Slot:          newSlot,
		Type:          asset.StatusEventTypeMoved,
		Body: asset.MovedStatusEventBody{
			OldSlot:   oldSlot,
			CreatedAt: createdAt,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func QuantityChangedEventStatusProvider(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, assetId uint32, templateId uint32, slot int16, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.QuantityChangedEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: compartmentId,
		AssetId:       assetId,
		TemplateId:    templateId,
		Slot:          slot,
		Type:          asset.StatusEventTypeQuantityChanged,
		Body: asset.QuantityChangedEventBody{
			Quantity: quantity,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func UpdatedEventStatusProvider(transactionId uuid.UUID, characterId uint32, a Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(a.Id()))
	value := &asset.StatusEvent[asset.UpdatedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: a.CompartmentId(),
		AssetId:       a.Id(),
		TemplateId:    a.TemplateId(),
		Slot:          a.Slot(),
		Type:          asset.StatusEventTypeUpdated,
		Body: asset.UpdatedStatusEventBody{
			AssetData: makeAssetData(a),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func AcceptedEventStatusProvider(transactionId uuid.UUID, characterId uint32, a Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(a.Id()))
	value := &asset.StatusEvent[asset.AcceptedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: a.CompartmentId(),
		AssetId:       a.Id(),
		TemplateId:    a.TemplateId(),
		Slot:          a.Slot(),
		Type:          asset.StatusEventTypeAccepted,
		Body: asset.AcceptedStatusEventBody{
			AssetData: makeAssetData(a),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ReleasedEventStatusProvider(transactionId uuid.UUID, characterId uint32, a Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(a.Id()))
	value := &asset.StatusEvent[asset.ReleasedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: a.CompartmentId(),
		AssetId:       a.Id(),
		TemplateId:    a.TemplateId(),
		Slot:          a.Slot(),
		Type:          asset.StatusEventTypeReleased,
		Body: asset.ReleasedStatusEventBody{
			AssetData: makeAssetData(a),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func ExpiredEventStatusProvider(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, assetId uint32, templateId uint32, slot int16, isCash bool, replaceItemId uint32, replaceMessage string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.ExpiredStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: compartmentId,
		AssetId:       assetId,
		TemplateId:    templateId,
		Slot:          slot,
		Type:          asset.StatusEventTypeExpired,
		Body: asset.ExpiredStatusEventBody{
			IsCash:         isCash,
			ReplaceItemId:  replaceItemId,
			ReplaceMessage: replaceMessage,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
