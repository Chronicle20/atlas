package item

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		itemId      uint32
		compartment Compartment
		subcategory string
	}{
		{1002000, CompartmentEquipment, "hat"},
		{1040002, CompartmentEquipment, "top"},
		{1041002, CompartmentEquipment, "top"},
		{1042000, CompartmentEquipment, "earring"},
		{1050000, CompartmentEquipment, "top"},
		{1060000, CompartmentEquipment, "bottom"},
		{1070000, CompartmentEquipment, "shoes"},
		{1080000, CompartmentEquipment, "gloves"},
		{1092000, CompartmentEquipment, "shield"},
		{1102000, CompartmentEquipment, "cape"},
		{1112000, CompartmentEquipment, "ring"},
		{1122000, CompartmentEquipment, "pendant"},
		{1132000, CompartmentEquipment, "belt"},
		{1142000, CompartmentEquipment, "medal"},
		{1900000, CompartmentEquipment, "tamed-mob"},
		{1910000, CompartmentEquipment, "saddle"},
		{1802000, CompartmentEquipment, "pet-equip"},
		{1302000, CompartmentEquipment, "one-handed-sword"},
		{1372000, CompartmentEquipment, "wand"},
		{1452000, CompartmentEquipment, "bow"},
		{1462000, CompartmentEquipment, "crossbow"},
		{1472000, CompartmentEquipment, "claw"},
		{2000000, CompartmentUse, "potion"},
		{2049000, CompartmentUse, "scroll"},
		{2070000, CompartmentUse, "throwing-star"},
		{2080000, CompartmentUse, "megaphone"},
		{2330000, CompartmentUse, "bullet"},
		{3010000, CompartmentSetup, "chair"},
		{3030000, CompartmentSetup, "hired-merchant"},
		{4000000, CompartmentEtc, "crafting-material"},
		{4010000, CompartmentEtc, "ore"},
		{4030000, CompartmentEtc, "production-item"},
		{4110000, CompartmentEtc, "monster-drop"},
		{4220000, CompartmentEtc, "quest-item"},
		{5000000, CompartmentCash, "pet"},
		{5040000, CompartmentCash, "teleport-rock"},
		{5072000, CompartmentCash, "megaphone"},
		{5140000, CompartmentCash, "store-permit"},
		{1990000, CompartmentEquipment, "other"},
		{2999000, CompartmentUse, "other"},
		{3999000, CompartmentSetup, "other-setup"},
		{4999000, CompartmentEtc, "other-etc"},
		{5999000, CompartmentCash, "other-cash"},
		{0, CompartmentUnknown, "other"},
		{9000000, CompartmentUnknown, "other"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("", func(t *testing.T) {
			gotComp, gotSub := Classify(tc.itemId)
			assert.Equalf(t, tc.compartment, gotComp, "itemId=%d compartment", tc.itemId)
			assert.Equalf(t, tc.subcategory, gotSub, "itemId=%d subcategory", tc.itemId)
		})
	}
}

func TestClassify_AllTaxonomyFixtures(t *testing.T) {
	cases := []struct {
		itemId      uint32
		compartment Compartment
		subcategory string
	}{
		{1002000, CompartmentEquipment, "hat"},
		{1040002, CompartmentEquipment, "top"},
		{1041002, CompartmentEquipment, "top"},
		{1452000, CompartmentEquipment, "bow"},
		{1462000, CompartmentEquipment, "crossbow"},
		{1472000, CompartmentEquipment, "claw"},
		{1302000, CompartmentEquipment, "one-handed-sword"},
		{1372000, CompartmentEquipment, "wand"},
		{2000000, CompartmentUse, "potion"},
		{2049000, CompartmentUse, "scroll"},
		{2070000, CompartmentUse, "throwing-star"},
		{2330000, CompartmentUse, "bullet"},
		{3010000, CompartmentSetup, "chair"},
		{3030000, CompartmentSetup, "hired-merchant"},
		{4000000, CompartmentEtc, "crafting-material"},
		{4010000, CompartmentEtc, "ore"},
		{4030000, CompartmentEtc, "production-item"},
		{5000000, CompartmentCash, "pet"},
		{5040000, CompartmentCash, "teleport-rock"},
		{5072000, CompartmentCash, "megaphone"},
		{5140000, CompartmentCash, "store-permit"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("", func(t *testing.T) {
			gotComp, gotSub := Classify(tc.itemId)
			assert.Equalf(t, tc.compartment, gotComp, "itemId=%d compartment", tc.itemId)
			assert.Equalf(t, tc.subcategory, gotSub, "itemId=%d subcategory", tc.itemId)
		})
	}
}
