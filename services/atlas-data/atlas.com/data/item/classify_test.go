package item

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/stretchr/testify/assert"
)

const compartmentUnknown = inventory.Type(0)

func TestClassify(t *testing.T) {
	cases := []struct {
		itemId      uint32
		compartment inventory.Type
		subcategory string
	}{
		{1002000, inventory.TypeValueEquip, "hat"},
		{1012070, inventory.TypeValueEquip, "face-accessory"},
		{1022000, inventory.TypeValueEquip, "eye-accessory"},
		{1032000, inventory.TypeValueEquip, "earring"},
		{1040002, inventory.TypeValueEquip, "top"},
		{1041002, inventory.TypeValueEquip, "top"},
		{1042000, inventory.TypeValueEquip, "top"},
		{1050000, inventory.TypeValueEquip, "overall"},
		{1060000, inventory.TypeValueEquip, "bottom"},
		{1070000, inventory.TypeValueEquip, "shoes"},
		{1080000, inventory.TypeValueEquip, "gloves"},
		{1092000, inventory.TypeValueEquip, "shield"},
		{1102000, inventory.TypeValueEquip, "cape"},
		{1112000, inventory.TypeValueEquip, "ring"},
		{1122000, inventory.TypeValueEquip, "pendant"},
		{1132000, inventory.TypeValueEquip, "belt"},
		{1142000, inventory.TypeValueEquip, "medal"},
		{1900000, inventory.TypeValueEquip, "tamed-mob"},
		{1910000, inventory.TypeValueEquip, "saddle"},
		{1802000, inventory.TypeValueEquip, "pet-equip"},
		{1302000, inventory.TypeValueEquip, "one-handed-sword"},
		{1372000, inventory.TypeValueEquip, "wand"},
		{1452000, inventory.TypeValueEquip, "bow"},
		{1462000, inventory.TypeValueEquip, "crossbow"},
		{1472000, inventory.TypeValueEquip, "claw"},
		{2000000, inventory.TypeValueUse, "potion"},
		{2049000, inventory.TypeValueUse, "scroll"},
		{2070000, inventory.TypeValueUse, "throwing-star"},
		{2080000, inventory.TypeValueUse, "megaphone"},
		{2330000, inventory.TypeValueUse, "bullet"},
		{3010000, inventory.TypeValueSetup, "chair"},
		{3030000, inventory.TypeValueSetup, "hired-merchant"},
		{4000000, inventory.TypeValueETC, "crafting-material"},
		{4010000, inventory.TypeValueETC, "ore"},
		{4030000, inventory.TypeValueETC, "production-item"},
		{4110000, inventory.TypeValueETC, "monster-drop"},
		{4220000, inventory.TypeValueETC, "quest-item"},
		{5000000, inventory.TypeValueCash, "pet"},
		{5040000, inventory.TypeValueCash, "teleport-rock"},
		{5072000, inventory.TypeValueCash, "megaphone"},
		{5140000, inventory.TypeValueCash, "store-permit"},
		{1990000, inventory.TypeValueEquip, "other"},
		{2999000, inventory.TypeValueUse, "other"},
		{3999000, inventory.TypeValueSetup, "other-setup"},
		{4999000, inventory.TypeValueETC, "other-etc"},
		{5999000, inventory.TypeValueCash, "other-cash"},
		{0, compartmentUnknown, "other"},
		{9000000, compartmentUnknown, "other"},
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
		compartment inventory.Type
		subcategory string
	}{
		{1002000, inventory.TypeValueEquip, "hat"},
		{1012070, inventory.TypeValueEquip, "face-accessory"},
		{1022000, inventory.TypeValueEquip, "eye-accessory"},
		{1032000, inventory.TypeValueEquip, "earring"},
		{1040002, inventory.TypeValueEquip, "top"},
		{1041002, inventory.TypeValueEquip, "top"},
		{1042000, inventory.TypeValueEquip, "top"},
		{1050000, inventory.TypeValueEquip, "overall"},
		{1452000, inventory.TypeValueEquip, "bow"},
		{1462000, inventory.TypeValueEquip, "crossbow"},
		{1472000, inventory.TypeValueEquip, "claw"},
		{1302000, inventory.TypeValueEquip, "one-handed-sword"},
		{1372000, inventory.TypeValueEquip, "wand"},
		{2000000, inventory.TypeValueUse, "potion"},
		{2049000, inventory.TypeValueUse, "scroll"},
		{2070000, inventory.TypeValueUse, "throwing-star"},
		{2330000, inventory.TypeValueUse, "bullet"},
		{3010000, inventory.TypeValueSetup, "chair"},
		{3030000, inventory.TypeValueSetup, "hired-merchant"},
		{4000000, inventory.TypeValueETC, "crafting-material"},
		{4010000, inventory.TypeValueETC, "ore"},
		{4030000, inventory.TypeValueETC, "production-item"},
		{5000000, inventory.TypeValueCash, "pet"},
		{5040000, inventory.TypeValueCash, "teleport-rock"},
		{5072000, inventory.TypeValueCash, "megaphone"},
		{5140000, inventory.TypeValueCash, "store-permit"},
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
