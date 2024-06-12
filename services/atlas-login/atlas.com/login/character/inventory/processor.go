package inventory

import (
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func GetEquippedItemsForCharacter(l logrus.FieldLogger, span opentracing.Span) func(characterId uint32) ([]EquippedItem, error) {
	return func(characterId uint32) ([]EquippedItem, error) {
		//	r, err := requestEquippedItemsForCharacter(characterId)(l, span)
		//	if err != nil {
		//		return nil, err
		//	}
		//
		//	eis := make([]EquippedItem, 0)
		//	for _, e := range requests.GetIncluded(r, equippedItemFilter) {
		//		ea, ok := requests.GetInclude[inventoryAttributes, equipmentStatisticsAttributes](r, e.EquipmentId)
		//		if ok {
		//			ei := NewEquippedItemBuilder().
		//				SetItemId(ea.ItemId).
		//				SetSlot(e.Slot).
		//				SetStrength(ea.Strength).
		//				SetDexterity(ea.Dexterity).
		//				SetIntelligence(ea.Intelligence).
		//				SetLuck(ea.Luck).
		//				SetHp(ea.Hp).
		//				SetMp(ea.Mp).
		//				SetWeaponAttack(ea.WeaponAttack).
		//				SetMagicAttack(ea.MagicAttack).
		//				SetWeaponDefense(ea.WeaponDefense).
		//				SetMagicDefense(ea.MagicDefense).
		//				SetAccuracy(ea.Accuracy).
		//				SetAvoidability(ea.Avoidability).
		//				SetHands(ea.Hands).
		//				SetSpeed(ea.Speed).
		//				SetJump(ea.Jump).
		//				SetSlots(ea.Slots).
		//				Build()
		//			eis = append(eis, ei)
		//		}
		//	}
		//
		//	return eis, nil
		return nil, nil
	}
}

//func equippedItemFilter(i requests.DataBody[equipmentAttributes]) bool {
//	attr := i.Attributes
//	return attr.Slot < 0
//}
