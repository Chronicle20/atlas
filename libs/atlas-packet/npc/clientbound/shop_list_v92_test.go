package clientbound

import (
	"bytes"
	"encoding/binary"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v92 NPC shop list. CShopDlg::SetShopDlg@0x6df6c0 (GMS_v92_1_DEVM.exe)
// reads Decode4 npcTemplateId, Decode2 count, then per item Decode4 itemId
// (v7@0x6df783), Decode4 mesoPrice (v95@0x6df79a), Decode1 discountRate
// (v96@0x6df7a5), FOUR Decode4 calls (v97@0x6df7b1, v98@0x6df7bb,
// v99@0x6df7c5, v100@0x6df7c8) — one more than v87's three
// (nTokenItemID/nItemPeriod/nLevelLimited @0x79e558-0x79e565, verified
// npc/clientbound/NpcShopList gms_v87) — confirming v92 already carries
// tokenTemplateId ahead of tokenPrice/period/levelLimit, then (rechargeable
// -> DecodeBuffer(8) unitPrice, else Decode2 quantity @0x6df815/0x6df806),
// Decode2 slotMax@0x6df822. Because v87 and v92 are adjacent columns in the
// coverage matrix (no IDB exists for the intervening 88-91 range), this
// pins the tokenTemplateId gate boundary at >=92 (shop_list.go), not the
// previously-unverified >=95.
//
// packet-audit:verify packet=npc/clientbound/NpcShopList version=gms_v92 ida=0x6df6c0
func TestNPCShopListV92(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 92, 1)
	shop := NewNPCShop(9010000, []ShopCommodity{
		{TemplateId: 2000000, MesoPrice: 50, DiscountRate: 5, TokenTemplateId: 4000000, TokenPrice: 10, Period: 0, LevelLimit: 30, IsAmmo: false, Quantity: 100, SlotMax: 200},
	})
	got := shop.Encode(l, ctx)(nil)

	le16 := func(v uint16) []byte { b := make([]byte, 2); binary.LittleEndian.PutUint16(b, v); return b }
	le32 := func(v uint32) []byte { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, v); return b }
	var want []byte
	want = append(want, le32(9010000)...) // npcTemplateId
	want = append(want, le16(1)...)       // count
	want = append(want, le32(2000000)...) // itemId
	want = append(want, le32(50)...)      // mesoPrice
	want = append(want, byte(5))          // discountRate (v87+)
	want = append(want, le32(4000000)...) // tokenTemplateId (v92+)
	want = append(want, le32(10)...)      // tokenPrice
	want = append(want, le32(0)...)       // period
	want = append(want, le32(30)...)      // levelLimit
	want = append(want, le16(100)...)     // quantity (not rechargeable)
	want = append(want, le16(200)...)     // maxPerSlot
	if !bytes.Equal(got, want) {
		t.Fatalf("v92 ShopList: got % x, want % x", got, want)
	}
}
