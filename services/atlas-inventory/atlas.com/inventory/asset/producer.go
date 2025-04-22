package asset

import (
	"atlas-inventory/kafka/message/asset"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func CreatedEventStatusProvider(characterId uint32, a Model[any]) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(a.Id()))
	value := &asset.StatusEvent[asset.CreatedStatusEventBody[any]]{
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

func DeletedEventStatusProvider(characterId uint32, compartmentId uuid.UUID, assetId uint32, templateId uint32, slot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.DeletedStatusEventBody]{
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

func MovedEventStatusProvider(characterId uint32, compartmentId uuid.UUID, assetId uint32, templateId uint32, newSlot int16, oldSlot int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.MovedStatusEventBody]{
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

func QuantityChangedEventStatusProvider(characterId uint32, compartmentId uuid.UUID, assetId uint32, templateId uint32, slot int16, quantity uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(assetId))
	value := &asset.StatusEvent[asset.QuantityChangedEventBody]{
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

func UpdatedEventStatusProvider(characterId uint32, a Model[any]) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(a.Id()))
	value := &asset.StatusEvent[asset.UpdatedStatusEventBody[any]]{
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
			Strength:       erd.Strength(),
			Dexterity:      erd.Dexterity(),
			Intelligence:   erd.Intelligence(),
			Luck:           erd.Luck(),
			Hp:             erd.HP(),
			Mp:             erd.MP(),
			WeaponAttack:   erd.WeaponAttack(),
			MagicAttack:    erd.MagicAttack(),
			WeaponDefense:  erd.WeaponDefense(),
			MagicDefense:   erd.MagicDefense(),
			Accuracy:       erd.Accuracy(),
			Avoidability:   erd.Avoidability(),
			Hands:          erd.Hands(),
			Speed:          erd.Speed(),
			Jump:           erd.Jump(),
			Slots:          erd.Slots(),
			OwnerId:        erd.OwnerId(),
			Locked:         erd.IsLocked(),
			Spikes:         erd.HasSpikes(),
			KarmaUsed:      erd.IsKarmaUsed(),
			Cold:           erd.IsCold(),
			CanBeTraded:    erd.CanBeTraded(),
			LevelType:      erd.LevelType(),
			Level:          erd.Level(),
			Experience:     erd.Experience(),
			HammersApplied: erd.HammersApplied(),
			Expiration:     erd.Expiration(),
		}
	}
	// TODO CashEquipableReferenceData
	if crd, ok := data.(ConsumableReferenceData); ok {
		return asset.ConsumableReferenceData{
			Quantity:     crd.Quantity(),
			OwnerId:      crd.OwnerId(),
			Flag:         crd.Flag(),
			Rechargeable: crd.Rechargeable(),
		}
	}
	if srd, ok := data.(SetupReferenceData); ok {
		return asset.SetupReferenceData{
			Quantity: srd.Quantity(),
			OwnerId:  srd.OwnerId(),
			Flag:     srd.Flag(),
		}
	}
	if trd, ok := data.(EtcReferenceData); ok {
		return asset.EtcReferenceData{
			Quantity: trd.Quantity(),
			OwnerId:  trd.OwnerId(),
			Flag:     trd.Flag(),
		}
	}
	if crd, ok := data.(CashReferenceData); ok {
		return asset.CashReferenceData{
			CashId:      crd.CashId(),
			Quantity:    crd.Quantity(),
			OwnerId:     crd.OwnerId(),
			Flag:        crd.Flag(),
			PurchasedBy: crd.PurchaseBy(),
		}
	}
	if prd, ok := data.(PetReferenceData); ok {
		return asset.PetReferenceData{
			CashId:      prd.CashId(),
			OwnerId:     prd.OwnerId(),
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
