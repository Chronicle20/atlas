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

// BUY / BUY-NOW / CANCEL / TAKE-HOME / PLACE-BID arms, gms_v95 opcode 0x134/308
// (GMS_v95.0_U_DEVM.exe, IDA port 13340). v95's PDB symbols expose these as
// named CITC::On* functions (inlined/unnamed on v83). Each arm references the
// listing by its ITC serial (nITCSN); only place-bid carries extra scalars. The
// item-slot blob is NOT present in these arms (unlike the register/sale arms).

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuy version=gms_v95 ida=0x573270
//
// ITC_OPERATION mode 0x10 buy-fixed-price. Derived from CITC::OnBuy @0x573270
// (COutPacket(308) @0x5732a5). Encode order @0x5732b8..0x5732cc:
//
//	Encode1(0x10u)         @0x5732b8  mode byte
//	Encode4(ii->p->nITCSN) @0x5732cc  itcSn
func TestItcOperationBuyByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationBuy(0x10, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x10)            // Encode1(0x10u) mode byte @0x5732b8
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5732cc
	if !bytes.Equal(got, want) {
		t.Fatalf("Buy (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyAuctionImm version=gms_v95 ida=0x573310
//
// ITC_OPERATION mode 0x14 buy-now-on-auction. Derived from
// CITC::OnBuyAuctionImm @0x573310 (COutPacket(308) @0x573345). Encode order
// @0x573358..0x57336c:
//
//	Encode1(0x14u)         @0x573358  mode byte
//	Encode4(ii->p->nITCSN) @0x57336c  itcSn
func TestItcOperationBuyAuctionImmByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationBuyAuctionImm(0x14, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x14)            // Encode1(0x14u) mode byte @0x573358
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x57336c
	if !bytes.Equal(got, want) {
		t.Fatalf("BuyAuctionImm (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelSale version=gms_v95 ida=0x5737a0
//
// ITC_OPERATION mode 0x07 cancel-sale. Derived from CITC::OnCancelSaleItem
// @0x5737a0 (COutPacket(308) @0x57381a). Encode order @0x57382d..0x57383d:
//
//	Encode1(7u)            @0x57382d  mode byte
//	Encode4(ii->p->nITCSN) @0x57383d  itcSn
func TestItcOperationCancelSaleByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationCancelSale(0x07, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x07)            // Encode1(7u) mode byte @0x57382d
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x57383d
	if !bytes.Equal(got, want) {
		t.Fatalf("CancelSale (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationMoveLtoS version=gms_v95 ida=0x573880
//
// ITC_OPERATION mode 0x08 take-home (move purchase locker->slot). Derived from
// CITC::OnMoveITCPurchaseItemLtoS @0x573880 (COutPacket(308) @0x5738b5). The
// nTI/nPos args are NOT written. Encode order @0x5738c8..0x5738dc:
//
//	Encode1(8u)            @0x5738c8  mode byte
//	Encode4(ii->p->nITCSN) @0x5738dc  itcSn
func TestItcOperationMoveLtoSByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationMoveLtoS(0x08, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x08)            // Encode1(8u) mode byte @0x5738c8
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5738dc
	if !bytes.Equal(got, want) {
		t.Fatalf("MoveLtoS (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationPlaceBid version=gms_v95 ida=0x58eb50
//
// ITC_OPERATION mode 0x13 place-bid. Send inlined into
// CITCBidAuctionDlg::OnButtonClicked @0x58eb50 (nId==1 confirm-bid branch,
// COutPacket(308) @0x58eda1). Encode order @0x58edb4..0x58ede7:
//
//	Encode1(0x13u)                @0x58edb4  mode byte
//	Encode4(m_pITCItem.p->nITCSN) @0x58edc7  itcSn
//	Encode4(m_nMyBidPrice)        @0x58edd7  bidPrice
//	Encode4(m_nMyBidRange)        @0x58ede7  bidRange
func TestItcOperationPlaceBidByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationPlaceBid(0x13, 123456, 5000, 100)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x13)            // Encode1(0x13u) mode byte @0x58edb4
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x58edc7
	want = append(want, le32(5000)...)   // Encode4 m_nMyBidPrice @0x58edd7
	want = append(want, le32(100)...)    // Encode4 m_nMyBidRange @0x58ede7
	if !bytes.Equal(got, want) {
		t.Fatalf("PlaceBid (v95):\n got %v\nwant %v", got, want)
	}
}

func TestItcOperationV95ArmsRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	t.Run("Buy", func(t *testing.T) {
		in := NewItcOperationBuy(0x10, 123456)
		out := ItcOperationBuy{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("BuyAuctionImm", func(t *testing.T) {
		in := NewItcOperationBuyAuctionImm(0x14, 123456)
		out := ItcOperationBuyAuctionImm{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("CancelSale", func(t *testing.T) {
		in := NewItcOperationCancelSale(0x07, 123456)
		out := ItcOperationCancelSale{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("MoveLtoS", func(t *testing.T) {
		in := NewItcOperationMoveLtoS(0x08, 123456)
		out := ItcOperationMoveLtoS{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("PlaceBid", func(t *testing.T) {
		in := NewItcOperationPlaceBid(0x13, 123456, 5000, 100)
		out := ItcOperationPlaceBid{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
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
