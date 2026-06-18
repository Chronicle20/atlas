package serverbound

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CORE-TRADE arms of the serverbound ITC_OPERATION mode-dispatcher, gms_v83
// opcode 0xFD/253 (MapleStory_dump.exe v83 Me, IDA port 13342). The dispatcher
// is one opcode + a leading Encode1(mode) byte selecting the marketplace
// operation. Per the dispatcher-family rule each arm gets its OWN byte fixture
// (enumerating mode bytes is a false pass).
//
// The leading mode byte and the trailing scalar fields are hand-computed from
// the cited COutPacket::Encode* calls; the embedded item-slot blob is produced
// by the shared, already-verified model.Asset codec (the GW_ItemSlotBase
// contract that sub_4E33D8 @0x4e33d8 writes: Encode1 type + virtual RawEncode).
// We assert the full byte sequence == modeByte || assetBytes || trailer, so the
// mode byte position and every trailer byte are pinned while the asset blob is
// delegated to its own verified codec.

// itcTestAsset is a concrete GW_ItemSlotBase fixture mirroring the clientbound
// MTS tests (model.NewAsset(true, 0, 2000000, zero).SetStackableInfo(5,0,0)).
func itcTestAsset() model.Asset {
	return model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(5, 0, 0)
}

func le32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterSale version=gms_v83 ida=0x59ec36
//
// ITC_OPERATION mode 2 register-fixed-price. Derived from
// CITC::OnRegisterSaleEntry @0x59ec36 (COutPacket(0xFD) @0x59ec63), arg0==0
// branch. Encode order @0x59ed92..0x59edd5:
//
//	Encode1(2u)          @0x59ed92  mode byte
//	sub_4E33D8(a5,&pkt)  @0x59ed9e  item-slot blob (model.Asset)
//	Encode4(a4)          @0x59eda9  quantity
//	Encode4(v22)         @0x59edb4  commodityId
//	Encode4(a3)          @0x59edbf  price
//	Encode1(a2[0])       @0x59edca  type
//	Encode1(v21[0])      @0x59edd5  flag
func TestItcOperationRegisterSaleByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationRegisterSale(0x02, asset, 5, 2000000, 1000, 0x01, 0x00)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x02)             // Encode1(2u) mode byte @0x59ed92
	want = append(want, assetBytes...)    // sub_4E33D8 item-slot blob @0x59ed9e
	want = append(want, le32(5)...)       // Encode4 quantity @0x59eda9
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x59edb4
	want = append(want, le32(1000)...)    // Encode4 price @0x59edbf
	want = append(want, 0x01)             // Encode1 type @0x59edca
	want = append(want, 0x00)             // Encode1 flag @0x59edd5
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterSale (v83):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterAuction version=gms_v83 ida=0x59ecc8
//
// ITC_OPERATION mode 0x12 register-auction. Derived from
// CITC::OnRegisterSaleEntry @0x59ec36 (COutPacket(0xFD) @0x59ec63), arg0==1
// branch. Encode order @0x59ecc8..0x59ed21:
//
//	Encode1(0x12u)       @0x59ecc8  mode byte
//	sub_4E33D8(a5,&pkt)  @0x59ecd4  item-slot blob (model.Asset)
//	Encode4(a4)          @0x59ecdf  quantity
//	Encode4(v22)         @0x59ecea  commodityId
//	Encode4(arg0)        @0x59ecf5  arg0 (==1 selector echo)
//	Encode4(v20)         @0x59ed00  buyNowPrice
//	Encode1(a2[0])       @0x59ed0b  type
//	Encode1(v21[0])      @0x59ed16  flag
//	Encode4(v19)         @0x59ed21  durationHrs
func TestItcOperationRegisterAuctionByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationRegisterAuction(0x12, asset, 5, 2000000, 1, 5000, 0x01, 0x00, 24)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x12)             // Encode1(0x12u) mode byte @0x59ecc8
	want = append(want, assetBytes...)    // sub_4E33D8 item-slot blob @0x59ecd4
	want = append(want, le32(5)...)       // Encode4 quantity @0x59ecdf
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x59ecea
	want = append(want, le32(1)...)       // Encode4 arg0 selector @0x59ecf5
	want = append(want, le32(5000)...)    // Encode4 buyNowPrice @0x59ed00
	want = append(want, 0x01)             // Encode1 type @0x59ed0b
	want = append(want, 0x00)             // Encode1 flag @0x59ed16
	want = append(want, le32(24)...)      // Encode4 durationHrs @0x59ed21
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterAuction (v83):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSaleCurrentItem version=gms_v83 ida=0x59ee3f
//
// ITC_OPERATION mode 3 sell-currently-selected-item. Derived from
// CITC::OnSaleCurrentItem @0x59ee3f (COutPacket(253) @0x59ee5d). Encode order
// @0x59ee6b..0x59ee98:
//
//	Encode1(3u)          @0x59ee6b  mode byte
//	Encode1(a2)          @0x59ee76  type
//	Encode4(a3)          @0x59ee81  slotPos
//	sub_4E33D8(a4,&pkt)  @0x59ee8d  item-slot blob (model.Asset)
//	Encode4(a5)          @0x59ee98  commodityId
func TestItcOperationSaleCurrentItemByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationSaleCurrentItem(0x03, 0x01, 7, asset, 2000000)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x03)             // Encode1(3u) mode byte @0x59ee6b
	want = append(want, 0x01)             // Encode1 type @0x59ee76
	want = append(want, le32(7)...)       // Encode4 slotPos @0x59ee81
	want = append(want, assetBytes...)    // sub_4E33D8 item-slot blob @0x59ee8d
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x59ee98
	if !bytes.Equal(got, want) {
		t.Fatalf("SaleCurrentItem (v83):\n got %v\nwant %v", got, want)
	}
}

func TestItcOperationCoreTradeRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	t.Run("RegisterSale", func(t *testing.T) {
		in := NewItcOperationRegisterSale(0x02, itcTestAsset(), 5, 2000000, 1000, 0x01, 0x00)
		out := ItcOperationRegisterSale{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("RegisterAuction", func(t *testing.T) {
		in := NewItcOperationRegisterAuction(0x12, itcTestAsset(), 5, 2000000, 1, 5000, 0x01, 0x00, 24)
		out := ItcOperationRegisterAuction{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("SaleCurrentItem", func(t *testing.T) {
		in := NewItcOperationSaleCurrentItem(0x03, 0x01, 7, itcTestAsset(), 2000000)
		out := ItcOperationSaleCurrentItem{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
}
