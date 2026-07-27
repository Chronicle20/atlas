package clientbound

import (
	"bytes"
	"encoding/binary"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v79 NPC shop list. CShopDlg::SetShopDlg@0x6d3459 (GMS_v79_1_DEVM.exe, port
// 13340) reads Decode4 npcTemplateId, Decode2 count, then per item Decode4
// itemId, Decode4 mesoPrice, (rechargeable -> DecodeBuffer(8) unitPrice, else
// Decode2 quantity), Decode2 maxPerSlot. The tokenPrice/period/levelLimit ints
// present at v83+ are absent in v79.
//
// packet-audit:verify packet=npc/clientbound/NpcShopList version=gms_v79 ida=0x6d3459
func TestNPCShopListV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 79, 1)
	shop := NewNPCShop(9010000, []ShopCommodity{
		{TemplateId: 2000000, MesoPrice: 50, IsAmmo: false, Quantity: 100, SlotMax: 200},
	})
	got := shop.Encode(l, ctx)(nil)

	le16 := func(v uint16) []byte { b := make([]byte, 2); binary.LittleEndian.PutUint16(b, v); return b }
	le32 := func(v uint32) []byte { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, v); return b }
	var want []byte
	want = append(want, le32(9010000)...) // npcTemplateId
	want = append(want, le16(1)...)       // count
	want = append(want, le32(2000000)...) // itemId
	want = append(want, le32(50)...)      // mesoPrice
	want = append(want, le16(100)...)     // quantity (not rechargeable)
	want = append(want, le16(200)...)     // maxPerSlot
	if !bytes.Equal(got, want) {
		t.Fatalf("v79 ShopList: got % x, want % x", got, want)
	}
}
