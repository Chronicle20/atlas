package clientbound

import (
	"bytes"
	"encoding/binary"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 NPC shop list. CShopDlg::SetShopDlg sub_5B430A@0x5b430a
// (GMS_v48_1_DEVM.exe, port 13337) reads Decode4 npcTemplateId, Decode2 count,
// then per item Decode4 itemId, Decode4 mesoPrice, [DecodeBuffer(8) unitPrice
// IF itemId/10000==207 rechargeable], Decode2 quantity — and NO trailing
// maxPerSlot short. The per-slot max was added between v48 and v61 (v61
// SetShopDlg@0x6437e3 / v79 @0x6d3459 both read it); v48 omits it. The
// discountRate/token/tokenPrice/period/levelLimit fields are all absent (v48 is
// far below their >=87/>=95/>=83 gates).
//
// packet-audit:verify packet=npc/clientbound/NpcShopList version=gms_v48 ida=0x5b430a
func TestNPCShopListV48(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 48, 1)
	shop := NewNPCShop(9010000, []ShopCommodity{
		{TemplateId: 2000000, MesoPrice: 50, IsAmmo: false, Quantity: 100, SlotMax: 200},
	})
	got := shop.Encode(l, ctx)(nil)

	le16 := func(v uint16) []byte { b := make([]byte, 2); binary.LittleEndian.PutUint16(b, v); return b }
	le32 := func(v uint32) []byte { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, v); return b }
	var want []byte
	want = append(want, le32(9010000)...) // npcTemplateId (Decode4 @0x5b4325)
	want = append(want, le16(1)...)       // count (Decode2 @0x5b4338)
	want = append(want, le32(2000000)...) // itemId (Decode4 @0x5b435b)
	want = append(want, le32(50)...)      // mesoPrice (Decode4 @0x5b436b)
	want = append(want, le16(100)...)     // quantity (Decode2 @0x5b43af); NO maxPerSlot
	if !bytes.Equal(got, want) {
		t.Fatalf("v48 ShopList: got % x, want % x", got, want)
	}
}
