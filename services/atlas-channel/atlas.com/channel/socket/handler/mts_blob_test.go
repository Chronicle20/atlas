package handler

import (
	mtsholding "atlas-channel/mts/holding"
	mtslisting "atlas-channel/mts/listing"
	mtswish "atlas-channel/mts/wish"
	"context"
	"testing"

	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/sirupsen/logrus"
)

// stackableTypeByte is the GW_ItemSlotBase type discriminator for a bundle/
// stackable item: encodeStackableInfo writes WriteByte(2) as the FIRST byte of a
// BARE (zeroPosition) blob. Every ITCITEM blob the server sends in an MTS list
// must lead with this. A leading inventory-slot byte (zeroPosition=false) is the
// v83 client crash: GW_ItemSlotBase::Decode reads the type byte first with no
// slot prefix, so a slot byte is misread as the item type and the rest of the
// item mis-decodes, overrunning a later DecodeStr.
const stackableTypeByte = 0x02

func firstItcItemBlobByte(t *testing.T, mi fieldcb.MtsItem) byte {
	t.Helper()
	item := mi.Item()
	b := item.Encode(logrus.New(), context.Background())(map[string]interface{}{})
	if len(b) == 0 {
		t.Fatal("empty ITCITEM item blob")
	}
	return b[0]
}

// TestMtsItemBuildersEmitBareBlob guards EVERY ITCITEM builder against the
// zeroPosition=false regression. The browse/sale list (ToMtsItem), the wish list,
// and the purchase/holding list all feed the same client-side per-item decoder
// (CITC sub_5A2C0F), so any one of them shipping a slot-prefixed blob crashes the
// client. The holding/purchase variant was the regression that recurred once a
// cancelled listing populated the take-home list.
func TestMtsItemBuildersEmitBareBlob(t *testing.T) {
	const useItem = uint32(2030019) // Nautilus return scroll (USE / stackable)

	listingM, err := mtslisting.Extract(mtslisting.RestModel{ItcSn: 1, TemplateId: useItem, Quantity: 3, ListValue: 200})
	if err != nil {
		t.Fatalf("listing extract: %v", err)
	}
	holdingM, err := mtsholding.Extract(mtsholding.RestModel{ItcSn: 3, TemplateId: useItem, Quantity: 3})
	if err != nil {
		t.Fatalf("holding extract: %v", err)
	}
	wishM, err := mtswish.Extract(mtswish.RestModel{Serial: 7, ItemId: useItem})
	if err != nil {
		t.Fatalf("wish extract: %v", err)
	}

	cases := []struct {
		name string
		item fieldcb.MtsItem
	}{
		{"listing/sale (ToMtsItem)", mtsItemFromListing(listingM)},
		{"holding/purchase", mtsItemFromHolding(holdingM)},
		{"wish", mtsItemFromWish(wishM)},
	}
	for _, c := range cases {
		if got := firstItcItemBlobByte(t, c.item); got != stackableTypeByte {
			t.Errorf("%s: ITCITEM blob leads with 0x%02x, want bare type byte 0x%02x — a slot-prefixed (zeroPosition=false) blob crashes the v83 client", c.name, got, stackableTypeByte)
		}
	}
}
