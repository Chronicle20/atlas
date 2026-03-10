package npc

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestNPCShop(t *testing.T) {
	commodities := []ShopCommodity{
		{TemplateId: 2000000, MesoPrice: 50, DiscountRate: 0, TokenTemplateId: 0, TokenPrice: 0, Period: 0, LevelLimit: 0, IsAmmo: false, Quantity: 100, SlotMax: 200},
		{TemplateId: 2000001, MesoPrice: 100, DiscountRate: 5, TokenTemplateId: 4000000, TokenPrice: 10, Period: 0, LevelLimit: 30, IsAmmo: false, Quantity: 50, SlotMax: 100},
	}
	input := NewNPCShop(9010000, commodities)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := ShopList{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
