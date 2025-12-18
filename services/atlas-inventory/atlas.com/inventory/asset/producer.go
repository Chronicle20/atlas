package asset

import (
	"atlas-inventory/kafka/message/asset"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func CreatedEventStatusProvider(transactionId uuid.UUID, characterId uint32, a Model[any]) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(a.Id()))
	value := &asset.StatusEvent[asset.CreatedStatusEventBody[any]]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: a.CompartmentId(),
		AssetId:       a.Id(),
		TemplateId:    a.TemplateId(),
		Slot:          a.Slot(),
		Type:          asset.StatusEventTypeCreated,
		Body: asset.CreatedStatusEventBody[any]{
			ReferenceId:   a.ReferenceId(),
			ReferenceType: string(a.ReferenceType()),
			ReferenceData: getReferenceData(a.ReferenceData()),
			Expiration:    a.Expiration(),
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

func MovedEventStatusProvider(transactionId uuid.UUID, characterId uint32, compartmentId uuid.UUID, assetId uint32, templateId uint32, newSlot int16, oldSlot int16) model.Provider[[]kafka.Message] {
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
			OldSlot: oldSlot,
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

func UpdatedEventStatusProvider(transactionId uuid.UUID, characterId uint32, a Model[any]) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(a.Id()))
	value := &asset.StatusEvent[asset.UpdatedStatusEventBody[any]]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		CompartmentId: a.CompartmentId(),
		AssetId:       a.Id(),
		TemplateId:    a.TemplateId(),
		Slot:          a.Slot(),
		Type:          asset.StatusEventTypeUpdated,
		Body: asset.UpdatedStatusEventBody[any]{
			ReferenceId:   a.ReferenceId(),
			ReferenceType: string(a.ReferenceType()),
			ReferenceData: getReferenceData(a.ReferenceData()),
			Expiration:    a.Expiration(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func getReferenceData(data any) interface{} {
	if erd, ok := data.(EquipableReferenceData); ok {
		return asset.EquipableReferenceData{
			BaseData: asset.BaseData{
				OwnerId: erd.ownerId,
			},
			StatisticData: asset.StatisticData{
				Strength:      erd.strength,
				Dexterity:     erd.dexterity,
				Intelligence:  erd.intelligence,
				Luck:          erd.luck,
				Hp:            erd.hp,
				Mp:            erd.mp,
				WeaponAttack:  erd.weaponAttack,
				MagicAttack:   erd.magicAttack,
				WeaponDefense: erd.weaponDefense,
				MagicDefense:  erd.magicDefense,
				Accuracy:      erd.accuracy,
				Avoidability:  erd.avoidability,
				Hands:         erd.hands,
				Speed:         erd.speed,
				Jump:          erd.jump,
			},
			Slots:          erd.slots,
			Locked:         erd.locked,
			Spikes:         erd.spikes,
			KarmaUsed:      erd.karmaUsed,
			Cold:           erd.cold,
			CanBeTraded:    erd.canBeTraded,
			LevelType:      erd.levelType,
			Level:          erd.level,
			Experience:     erd.experience,
			HammersApplied: erd.hammersApplied,
		}
	}
	if crd, ok := data.(CashEquipableReferenceData); ok {
		return asset.CashEquipableReferenceData{
			CashData: asset.CashData{
				CashId: crd.cashId,
			},
		}
	}
	if crd, ok := data.(ConsumableReferenceData); ok {
		return asset.ConsumableReferenceData{
			BaseData: asset.BaseData{
				OwnerId: crd.ownerId,
			},
			StackableData: asset.StackableData{
				Quantity: crd.quantity,
			},
			Flag:         crd.Flag(),
			Rechargeable: crd.Rechargeable(),
		}
	}
	if srd, ok := data.(SetupReferenceData); ok {
		return asset.SetupReferenceData{
			BaseData: asset.BaseData{
				OwnerId: srd.ownerId,
			},
			StackableData: asset.StackableData{
				Quantity: srd.quantity,
			},
			Flag: srd.Flag(),
		}
	}
	if trd, ok := data.(EtcReferenceData); ok {
		return asset.EtcReferenceData{
			BaseData: asset.BaseData{
				OwnerId: trd.ownerId,
			},
			StackableData: asset.StackableData{
				Quantity: trd.quantity,
			},
			Flag: trd.Flag(),
		}
	}
	if crd, ok := data.(CashReferenceData); ok {
		return asset.CashReferenceData{
			BaseData: asset.BaseData{
				OwnerId: crd.ownerId,
			},
			StackableData: asset.StackableData{
				Quantity: crd.quantity,
			},
			CashData: asset.CashData{
				CashId: crd.cashId,
			},
			Flag:        crd.Flag(),
			PurchasedBy: crd.PurchaseBy(),
		}
	}
	if prd, ok := data.(PetReferenceData); ok {
		return asset.PetReferenceData{
			BaseData: asset.BaseData{
				OwnerId: prd.ownerId,
			},
			CashData: asset.CashData{
				CashId: prd.cashId,
			},
			Flag:        prd.Flag(),
			PurchasedBy: prd.PurchaseBy(),
			Name:        prd.Name(),
			Level:       prd.Level(),
			Closeness:   prd.Closeness(),
			Fullness:    prd.Fullness(),
			Slot:        prd.Slot(),
		}
	}
	return nil
}
