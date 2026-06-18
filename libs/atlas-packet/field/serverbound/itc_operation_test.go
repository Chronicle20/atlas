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

// WISH-LIST / ZZIM (favorite) arms, gms_v95 opcode 0x134/308
// (GMS_v95.0_U_DEVM.exe, IDA port 13340). v95's PDB symbols expose these as
// named CITC::On* functions. Six are serial-only (mode + Encode4(nITCSN)) like
// the buy/cancel arms; OnRegisterWishEntry carries a full wish-entry body.

// packet-audit:verify packet=field/serverbound/FieldItcOperationSetZzim version=gms_v95 ida=0x5733b0
//
// ITC_OPERATION mode 0x09 set-zzim (add to wishlist/favorite). Derived from
// CITC::OnSetZzim @0x5733b0 (COutPacket(308) @0x5733e5). Encode order
// @0x5733f8..0x57340c:
//
//	Encode1(9u)            @0x5733f8  mode byte
//	Encode4(ii->p->nITCSN) @0x57340c  itcSn
func TestItcOperationSetZzimByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationSetZzim(0x09, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x09)            // Encode1(9u) mode byte @0x5733f8
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x57340c
	if !bytes.Equal(got, want) {
		t.Fatalf("SetZzim (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyZzim version=gms_v95 ida=0x573450
//
// ITC_OPERATION mode 0x11 buy-zzim (buy a favorited item). Derived from
// CITC::OnBuyZzim @0x573450 (COutPacket(308) @0x5734b7). A YesNo confirm
// gates the send; it does not change the wire. Encode order @0x5734ca..0x5734de:
//
//	Encode1(0x11u)         @0x5734ca  mode byte
//	Encode4(ii->p->nITCSN) @0x5734de  itcSn
func TestItcOperationBuyZzimByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationBuyZzim(0x11, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x11)            // Encode1(0x11u) mode byte @0x5734ca
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5734de
	if !bytes.Equal(got, want) {
		t.Fatalf("BuyZzim (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationDeleteZzim version=gms_v95 ida=0x573520
//
// ITC_OPERATION mode 0x0A delete-zzim (remove favorite). Derived from
// CITC::OnDeleteZzim @0x573520 (COutPacket(308) @0x573555). Encode order
// @0x573568..0x57357c:
//
//	Encode1(0xAu)          @0x573568  mode byte
//	Encode4(ii->p->nITCSN) @0x57357c  itcSn
func TestItcOperationDeleteZzimByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationDeleteZzim(0x0A, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0A)            // Encode1(0xAu) mode byte @0x573568
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x57357c
	if !bytes.Equal(got, want) {
		t.Fatalf("DeleteZzim (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationViewWish version=gms_v95 ida=0x5735c0
//
// ITC_OPERATION mode 0x0B view-wish. Derived from CITC::OnViewWish @0x5735c0
// (COutPacket(308) @0x5735f5). Encode order @0x573608..0x57361c:
//
//	Encode1(0xBu)          @0x573608  mode byte
//	Encode4(ii->p->nITCSN) @0x57361c  itcSn
func TestItcOperationViewWishByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationViewWish(0x0B, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0B)            // Encode1(0xBu) mode byte @0x573608
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x57361c
	if !bytes.Equal(got, want) {
		t.Fatalf("ViewWish (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyWish version=gms_v95 ida=0x573660
//
// ITC_OPERATION mode 0x0C buy-wish. Derived from CITC::OnBuyWish @0x573660
// (COutPacket(308) @0x573695). Encode order @0x5736a8..0x5736bc:
//
//	Encode1(0xCu)          @0x5736a8  mode byte
//	Encode4(ii->p->nITCSN) @0x5736bc  itcSn
func TestItcOperationBuyWishByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationBuyWish(0x0C, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0C)            // Encode1(0xCu) mode byte @0x5736a8
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5736bc
	if !bytes.Equal(got, want) {
		t.Fatalf("BuyWish (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelWish version=gms_v95 ida=0x573700
//
// ITC_OPERATION mode 0x0D cancel-wish. Derived from CITC::OnCancelWish
// @0x573700 (COutPacket(308) @0x573735). Encode order @0x573748..0x57375c:
//
//	Encode1(0xDu)          @0x573748  mode byte
//	Encode4(ii->p->nITCSN) @0x57375c  itcSn
func TestItcOperationCancelWishByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationCancelWish(0x0D, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0D)            // Encode1(0xDu) mode byte @0x573748
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x57375c
	if !bytes.Equal(got, want) {
		t.Fatalf("CancelWish (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterWishEntry version=gms_v95 ida=0x573c10
//
// ITC_OPERATION mode 0x04 register-wish-entry. Derived from
// CITC::OnRegisterWishEntry @0x573c10 (COutPacket(308) @0x573ca5). A 110-NX
// floor guard gates a notice; it does not change the wire. Encode order
// @0x573cb5..0x573d23:
//
//	Encode1(4u)                           @0x573cb5  mode byte
//	Encode4(m_nWishItemID)                @0x573cc5  itemId
//	Encode4(m_nWishPrice)                 @0x573cd5  price
//	Encode4(m_nWishCount)                 @0x573ce5  count
//	Encode1(m_bWishDuration)              @0x573cf6  duration
//	Encode1(m_bWishRegistrationFeeOption) @0x573d07  feeOption
//	EncodeStr(m_sWishDesc)                @0x573d23  description (uint16 len + bytes)
func TestItcOperationRegisterWishEntryByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationRegisterWishEntry(0x04, 2000000, 1000, 5, 0x07, 0x01, "wish")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x04)              // Encode1(4u) mode byte @0x573cb5
	want = append(want, le32(2000000)...)  // Encode4 itemId @0x573cc5
	want = append(want, le32(1000)...)     // Encode4 price @0x573cd5
	want = append(want, le32(5)...)        // Encode4 count @0x573ce5
	want = append(want, 0x07)              // Encode1 duration @0x573cf6
	want = append(want, 0x01)              // Encode1 feeOption @0x573d07
	want = append(want, 0x04, 0x00)        // EncodeStr len prefix (uint16 LE = 4) @0x573d23
	want = append(want, []byte("wish")...) // EncodeStr bytes @0x573d23
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterWishEntry (v95):\n got %v\nwant %v", got, want)
	}
}

// REGISTER / SALE / AUCTION arms, gms_v95 opcode 0x134/308 (GMS_v95.0_U_DEVM.exe,
// IDA port 13340). The structs were verified for v83 (CITC::OnRegisterSaleEntry /
// OnSaleCurrentItem). The v95 decompiles (OnRegisterSaleEntry @0x572e90,
// OnSaleCurrentItem @0x5731a0) confirm the SAME mode bytes + body shapes — only
// the dispatcher opcode differs (v83 0xFD/253 vs v95 0x134/308). These fixtures
// pin the v95 coverage; the embedded item-slot blob delegates to the verified
// model.Asset codec (GW_ItemSlotBase::Encode @0x4f6660).

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterSale version=gms_v95 ida=0x572e90
//
// ITC_OPERATION mode 2 register-fixed-price. Derived from
// CITC::OnRegisterSaleEntry @0x572e90 (COutPacket(308) @0x572f53), nRegType==0
// branch (CRegisterSaleEntryDlg). Encode order @0x5730c9..0x573117:
//
//	Encode1(2u)              @0x5730c9  mode byte
//	GW_ItemSlotBase::Encode  @0x5730d5  item-slot blob (model.Asset)
//	Encode4(v6=nSlotNo)      @0x5730df  quantity
//	Encode4(nSaleCount)      @0x5730ed  commodityId
//	Encode4(nSlotNo)         @0x5730fb  price
//	Encode1(pItem)           @0x573109  type
//	Encode1(nRegFeeOption)   @0x573117  flag
func TestItcOperationRegisterSaleByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationRegisterSale(0x02, asset, 5, 2000000, 1000, 0x01, 0x00)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x02)             // Encode1(2u) mode byte @0x5730c9
	want = append(want, assetBytes...)    // GW_ItemSlotBase::Encode item-slot blob @0x5730d5
	want = append(want, le32(5)...)       // Encode4 quantity @0x5730df
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x5730ed
	want = append(want, le32(1000)...)    // Encode4 price @0x5730fb
	want = append(want, 0x01)             // Encode1 type @0x573109
	want = append(want, 0x00)             // Encode1 flag @0x573117
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterSale (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterAuction version=gms_v95 ida=0x572fd0
//
// ITC_OPERATION mode 0x12 register-auction. Derived from
// CITC::OnRegisterSaleEntry @0x572e90 (COutPacket(308) @0x572f53), nRegType==1
// branch (CRegisterAuctionEntryDlg). Encode order @0x572fd0..0x57303a:
//
//	Encode1(0x12u)           @0x572fd0  mode byte
//	GW_ItemSlotBase::Encode  @0x572fdc  item-slot blob (model.Asset)
//	Encode4(v6=nSlotNo)      @0x572fe6  quantity
//	Encode4(nSaleCount)      @0x572ff4  commodityId
//	Encode4(nTI)             @0x573002  selector
//	Encode4(nRegType)        @0x573010  buyNowPrice
//	Encode1(pItem)           @0x57301e  type
//	Encode1(nRegFeeOption)   @0x57302c  flag
//	Encode4(nBidRange)       @0x57303a  durationHrs
func TestItcOperationRegisterAuctionByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationRegisterAuction(0x12, asset, 5, 2000000, 1, 5000, 0x01, 0x00, 24)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x12)             // Encode1(0x12u) mode byte @0x572fd0
	want = append(want, assetBytes...)    // GW_ItemSlotBase::Encode item-slot blob @0x572fdc
	want = append(want, le32(5)...)       // Encode4 quantity @0x572fe6
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x572ff4
	want = append(want, le32(1)...)       // Encode4 selector @0x573002
	want = append(want, le32(5000)...)    // Encode4 buyNowPrice @0x573010
	want = append(want, 0x01)             // Encode1 type @0x57301e
	want = append(want, 0x00)             // Encode1 flag @0x57302c
	want = append(want, le32(24)...)      // Encode4 durationHrs @0x57303a
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterAuction (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSaleCurrentItem version=gms_v95 ida=0x5731a0
//
// ITC_OPERATION mode 3 sell-currently-selected-item. Derived from
// CITC::OnSaleCurrentItem @0x5731a0 (COutPacket(308) @0x5731d9). Encode order
// @0x5731ec..0x573224:
//
//	Encode1(3u)              @0x5731ec  mode byte
//	Encode1(nItemTI)         @0x5731fa  type
//	Encode4(nSlotPosition)   @0x573208  slotPos
//	GW_ItemSlotBase::Encode  @0x573216  item-slot blob (model.Asset)
//	Encode4(nITCSN)          @0x573224  commodityId
func TestItcOperationSaleCurrentItemByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationSaleCurrentItem(0x03, 0x01, 7, asset, 2000000)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x03)             // Encode1(3u) mode byte @0x5731ec
	want = append(want, 0x01)             // Encode1 type @0x5731fa
	want = append(want, le32(7)...)       // Encode4 slotPos @0x573208
	want = append(want, assetBytes...)    // GW_ItemSlotBase::Encode item-slot blob @0x573216
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x573224
	if !bytes.Equal(got, want) {
		t.Fatalf("SaleCurrentItem (v95):\n got %v\nwant %v", got, want)
	}
}

// BROWSE-NAVIGATION arms, gms_v95 opcode 0x134/308 (GMS_v95.0_U_DEVM.exe, IDA
// port 13340). Three senders emit mode 0x05 (GetItcList browse request, 8-field
// shape); the tab search button emits mode 0x06 (6-field shape, no sort bytes).
// Each arm gets its own byte fixture per the dispatcher-family rule.

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategory version=gms_v95 ida=0x5744a0
//
// ITC_OPERATION mode 0x05 change-browse-category. Derived from
// CITC::OnChangedCategory @0x5744a0 (COutPacket(308) @0x57451a). Only nCategory
// is variable; sub/page/sort/option/cond are constants. Encode order
// @0x57452d..0x5745ac:
//
//	Encode1(5u)        @0x57452d  mode byte
//	Encode4(nCategory) @0x57453b  category
//	Encode4(0)         @0x574546  categorySub (const 0)
//	Encode4(0)         @0x574551  page (const 0)
//	Encode1(1u)        @0x57455c  sortType (const 1)
//	Encode1(1u)        @0x574567  sortColumn (const 1)
//	Encode4(1u)        @0x574572  searchOption (const 1)
//	EncodeStr("")      @0x5745ac  searchCondition (const empty; uint16 len 0)
func TestItcOperationChangedCategoryByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationChangedCategory(0x05, 3)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x05)       // Encode1(5u) mode byte @0x57452d
	want = append(want, le32(3)...) // Encode4 category @0x57453b
	want = append(want, le32(0)...) // Encode4 categorySub @0x574546
	want = append(want, le32(0)...) // Encode4 page @0x574551
	want = append(want, 0x01)       // Encode1 sortType @0x57455c
	want = append(want, 0x01)       // Encode1 sortColumn @0x574567
	want = append(want, le32(1)...) // Encode4 searchOption @0x574572
	want = append(want, 0x00, 0x00) // EncodeStr("") uint16 len 0 @0x5745ac
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedCategory (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategorySub version=gms_v95 ida=0x5739a0
//
// ITC_OPERATION mode 0x05 change-sub-category. Derived from
// CITC::OnChangedCategorySub @0x5739a0 (COutPacket(308) @0x5739da). page=0
// const; cat/sub/sort/option/cond from state/args. Encode order
// @0x5739ed..0x573aa0:
//
//	Encode1(5u)                  @0x5739ed  mode byte
//	Encode4(m_nCurCategory)      @0x5739fa  category
//	Encode4(nCategorySub)        @0x573a08  categorySub
//	Encode4(0)                   @0x573a13  page (const 0)
//	Encode1(nSortType)           @0x573a21  sortType
//	Encode1(nSortColumn)         @0x573a2f  sortColumn
//	Encode4(searchOption)        @0x573a3f/0x573a7e  searchOption
//	EncodeStr(searchCondition)   @0x573aa0  searchCondition
func TestItcOperationChangedCategorySubByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationChangedCategorySub(0x05, 3, 2, 0x01, 0x01, 1, "")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x05)       // Encode1(5u) mode byte @0x5739ed
	want = append(want, le32(3)...) // Encode4 category @0x5739fa
	want = append(want, le32(2)...) // Encode4 categorySub @0x573a08
	want = append(want, le32(0)...) // Encode4 page @0x573a13
	want = append(want, 0x01)       // Encode1 sortType @0x573a21
	want = append(want, 0x01)       // Encode1 sortColumn @0x573a2f
	want = append(want, le32(1)...) // Encode4 searchOption @0x573a3f
	want = append(want, 0x00, 0x00) // EncodeStr("") uint16 len 0 @0x573aa0
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedCategorySub (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedPage version=gms_v95 ida=0x573af0
//
// ITC_OPERATION mode 0x05 change-browse-page. Derived from CITC::OnChangedPage
// @0x573af0 (COutPacket(308) @0x573b29). All 8 fields from CITC state. Encode
// order @0x573b3c..0x573bb2:
//
//	Encode1(5u)                  @0x573b3c  mode byte
//	Encode4(m_nCurCategory)      @0x573b49  category
//	Encode4(m_nCurCategorySub)   @0x573b56  categorySub
//	Encode4(nPage)               @0x573b64  page
//	Encode1(m_nSortType)         @0x573b72  sortType
//	Encode1(m_nSortColumn)       @0x573b80  sortColumn
//	Encode4(m_nSearchOption)     @0x573b90  searchOption
//	EncodeStr(m_sSearchCondition)@0x573bb2  searchCondition
func TestItcOperationChangedPageByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationChangedPage(0x05, 3, 2, 4, 0x01, 0x01, 1, "abc")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x05)             // Encode1(5u) mode byte @0x573b3c
	want = append(want, le32(3)...)       // Encode4 category @0x573b49
	want = append(want, le32(2)...)       // Encode4 categorySub @0x573b56
	want = append(want, le32(4)...)       // Encode4 page @0x573b64
	want = append(want, 0x01)             // Encode1 sortType @0x573b72
	want = append(want, 0x01)             // Encode1 sortColumn @0x573b80
	want = append(want, le32(1)...)       // Encode4 searchOption @0x573b90
	want = append(want, 0x03, 0x00)       // EncodeStr len prefix (uint16 LE = 3) @0x573bb2
	want = append(want, []byte("abc")...) // EncodeStr bytes @0x573bb2
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedPage (v95):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationTabSearch version=gms_v95 ida=0x584b10
//
// ITC_OPERATION mode 0x06 tab/search-by-name. Send inlined into
// CITCWnd_Tab::OnButtonClicked @0x584b10 (nId==1004 search button, COutPacket(308)
// @0x584bc7/@0x584cc9). 6-field body, NO sort bytes (distinct from mode 0x05).
// Encode order @0x584bd7..0x584c1b (character-name branch shown; advanced-search
// branch @0x584cd9.. is byte-identical shape):
//
//	Encode1(6u)            @0x584bd7  mode byte
//	Encode4(m_nSelect+1)   @0x584be1  category
//	Encode4(m_nSelect)     @0x584beb  categorySub
//	Encode4(0)             @0x584bf5  page (const 0)
//	Encode4(searchOption)  @0x584bff  searchOption
//	EncodeStr(searchName)  @0x584c1b  searchCondition
func TestItcOperationTabSearchByteOutput_v95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewItcOperationTabSearch(0x06, 3, 2, 0, "hero")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x06)              // Encode1(6u) mode byte @0x584bd7
	want = append(want, le32(3)...)        // Encode4 category @0x584be1
	want = append(want, le32(2)...)        // Encode4 categorySub @0x584beb
	want = append(want, le32(0)...)        // Encode4 page @0x584bf5
	want = append(want, le32(0)...)        // Encode4 searchOption @0x584bff
	want = append(want, 0x04, 0x00)        // EncodeStr len prefix (uint16 LE = 4) @0x584c1b
	want = append(want, []byte("hero")...) // EncodeStr bytes @0x584c1b
	if !bytes.Equal(got, want) {
		t.Fatalf("TabSearch (v95):\n got %v\nwant %v", got, want)
	}
}

func TestItcOperationBrowseNavRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	t.Run("ChangedCategory", func(t *testing.T) {
		in := NewItcOperationChangedCategory(0x05, 3)
		out := ItcOperationChangedCategory{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("ChangedCategorySub", func(t *testing.T) {
		in := NewItcOperationChangedCategorySub(0x05, 3, 2, 0x01, 0x01, 1, "")
		out := ItcOperationChangedCategorySub{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("ChangedPage", func(t *testing.T) {
		in := NewItcOperationChangedPage(0x05, 3, 2, 4, 0x01, 0x01, 1, "abc")
		out := ItcOperationChangedPage{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("TabSearch", func(t *testing.T) {
		in := NewItcOperationTabSearch(0x06, 3, 2, 0, "hero")
		out := ItcOperationTabSearch{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
}

func TestItcOperationRegisterV95RoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
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
	t.Run("SetZzim", func(t *testing.T) {
		in := NewItcOperationSetZzim(0x09, 123456)
		out := ItcOperationSetZzim{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("BuyZzim", func(t *testing.T) {
		in := NewItcOperationBuyZzim(0x11, 123456)
		out := ItcOperationBuyZzim{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("DeleteZzim", func(t *testing.T) {
		in := NewItcOperationDeleteZzim(0x0A, 123456)
		out := ItcOperationDeleteZzim{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("ViewWish", func(t *testing.T) {
		in := NewItcOperationViewWish(0x0B, 123456)
		out := ItcOperationViewWish{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("BuyWish", func(t *testing.T) {
		in := NewItcOperationBuyWish(0x0C, 123456)
		out := ItcOperationBuyWish{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("CancelWish", func(t *testing.T) {
		in := NewItcOperationCancelWish(0x0D, 123456)
		out := ItcOperationCancelWish{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	})
	t.Run("RegisterWishEntry", func(t *testing.T) {
		in := NewItcOperationRegisterWishEntry(0x04, 2000000, 1000, 5, 0x07, 0x01, "wish")
		out := ItcOperationRegisterWishEntry{}
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

// =============================================================================
// gms_v87 — all 18 ITC_OPERATION serverbound arms, dispatcher opcode 0x10B/267
// (GMSv87_4GB.exe, IDA port 13341). v87 has full mangled CITC symbols (none
// inlined except PlaceBid/TabSearch, which inline into the dialog/tab button
// handlers — same as v95). The dispatcher opcode COutPacket(0x10B) is confirmed
// at CITC::OnRegisterSaleEntry @0x5cea27 (== registry opcode 267, NOT stale).
// EVERY arm's mode byte is BYTE-IDENTICAL to gms_v95 — no divergence. The item-
// slot blob is written by sub_502670 (v87's GW_ItemSlotBase writer; v83 used
// sub_4E33D8, v95 GW_ItemSlotBase::Encode) — same model.Asset contract. Per the
// dispatcher-family rule each arm gets its OWN byte fixture.
// =============================================================================

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterSale version=gms_v87 ida=0x5ce967
//
// ITC_OPERATION mode 2 register-fixed-price. Derived from
// CITC::OnRegisterSaleEntry @0x5ce967 (COutPacket(0x10B) @0x5cea27), arg0==0
// branch. Encode order @0x5ceb55..0x5ceb97:
//
//	Encode1(2u)          @0x5ceb55  mode byte
//	sub_502670(&pkt)     @0x5ceb60  item-slot blob (model.Asset)
//	Encode4(a4)          @0x5ceb6b  quantity
//	Encode4(v27)         @0x5ceb76  commodityId
//	Encode4(arg4)        @0x5ceb81  price
//	Encode1(a2)          @0x5ceb8c  type
//	Encode1(v26[0])      @0x5ceb97  flag
func TestItcOperationRegisterSaleByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationRegisterSale(0x02, asset, 5, 2000000, 1000, 0x01, 0x00)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x02)             // Encode1(2u) mode byte @0x5ceb55
	want = append(want, assetBytes...)    // sub_502670 item-slot blob @0x5ceb60
	want = append(want, le32(5)...)       // Encode4 quantity @0x5ceb6b
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x5ceb76
	want = append(want, le32(1000)...)    // Encode4 price @0x5ceb81
	want = append(want, 0x01)             // Encode1 type @0x5ceb8c
	want = append(want, 0x00)             // Encode1 flag @0x5ceb97
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterSale (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterAuction version=gms_v87 ida=0x5cea89
//
// ITC_OPERATION mode 0x12 register-auction. Derived from
// CITC::OnRegisterSaleEntry @0x5ce967 (COutPacket(0x10B) @0x5cea27), arg0==1
// branch. Encode order @0x5cea89..0x5ceae1:
//
//	Encode1(0x12u)       @0x5cea89  mode byte
//	sub_502670(&pkt)     @0x5cea94  item-slot blob (model.Asset)
//	Encode4(a4)          @0x5cea9f  quantity
//	Encode4(v27)         @0x5ceaaa  commodityId
//	Encode4(arg0)        @0x5ceab5  selector (==1)
//	Encode4(v25)         @0x5ceac0  buyNowPrice
//	Encode1(a2)          @0x5ceacb  type
//	Encode1(v26[0])      @0x5cead6  flag
//	Encode4(v24)         @0x5ceae1  durationHrs
func TestItcOperationRegisterAuctionByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationRegisterAuction(0x12, asset, 5, 2000000, 1, 5000, 0x01, 0x00, 24)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x12)             // Encode1(0x12u) mode byte @0x5cea89
	want = append(want, assetBytes...)    // sub_502670 item-slot blob @0x5cea94
	want = append(want, le32(5)...)       // Encode4 quantity @0x5cea9f
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x5ceaaa
	want = append(want, le32(1)...)       // Encode4 selector @0x5ceab5
	want = append(want, le32(5000)...)    // Encode4 buyNowPrice @0x5ceac0
	want = append(want, 0x01)             // Encode1 type @0x5ceacb
	want = append(want, 0x00)             // Encode1 flag @0x5cead6
	want = append(want, le32(24)...)      // Encode4 durationHrs @0x5ceae1
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterAuction (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSaleCurrentItem version=gms_v87 ida=0x5cec03
//
// ITC_OPERATION mode 3 sell-currently-selected-item. Derived from
// CITC::OnSaleCurrentItem @0x5cec03 (COutPacket(0x10B) @0x5cec21). Encode order
// @0x5cec2f..0x5cec5c:
//
//	Encode1(3u)          @0x5cec2f  mode byte
//	Encode1(a2)          @0x5cec3a  type
//	Encode4(arg4)        @0x5cec45  slotPos
//	sub_502670(&pkt)     @0x5cec51  item-slot blob (model.Asset)
//	Encode4(a5)          @0x5cec5c  commodityId
func TestItcOperationSaleCurrentItemByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationSaleCurrentItem(0x03, 0x01, 7, asset, 2000000)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x03)             // Encode1(3u) mode byte @0x5cec2f
	want = append(want, 0x01)             // Encode1 type @0x5cec3a
	want = append(want, le32(7)...)       // Encode4 slotPos @0x5cec45
	want = append(want, assetBytes...)    // sub_502670 item-slot blob @0x5cec51
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x5cec5c
	if !bytes.Equal(got, want) {
		t.Fatalf("SaleCurrentItem (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuy version=gms_v87 ida=0x5cf3fa
//
// ITC_OPERATION mode 0x10 buy-fixed-price. CITC::OnBuy @0x5cf3fa
// (COutPacket(0x10B) @0x5cf418). Encode1(0x10) @0x5cf426, Encode4 nITCSN @0x5cf437.
func TestItcOperationBuyByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationBuy(0x10, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x10)            // Encode1(0x10u) mode byte @0x5cf426
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5cf437
	if !bytes.Equal(got, want) {
		t.Fatalf("Buy (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyAuctionImm version=gms_v87 ida=0x5cf46d
//
// ITC_OPERATION mode 0x14 buy-now-on-auction. CITC::OnBuyAuctionImm @0x5cf46d
// (COutPacket(0x10B) @0x5cf48b). Encode1(0x14) @0x5cf499, Encode4 nITCSN @0x5cf4aa.
func TestItcOperationBuyAuctionImmByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationBuyAuctionImm(0x14, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x14)            // Encode1(0x14u) mode byte @0x5cf499
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5cf4aa
	if !bytes.Equal(got, want) {
		t.Fatalf("BuyAuctionImm (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelSale version=gms_v87 ida=0x5cf8dd
//
// ITC_OPERATION mode 0x07 cancel-sale. CITC::OnCancelSaleItem @0x5cf8dd
// (YesNo gate; suppressed if listing has bids; COutPacket(0x10B) @0x5cf933).
// Encode1(7) @0x5cf940, Encode4 nITCSN @0x5cf94e.
func TestItcOperationCancelSaleByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationCancelSale(0x07, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x07)            // Encode1(7u) mode byte @0x5cf940
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5cf94e
	if !bytes.Equal(got, want) {
		t.Fatalf("CancelSale (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationMoveLtoS version=gms_v87 ida=0x5cf986
//
// ITC_OPERATION mode 0x08 take-home. CITC::OnMoveITCPurchaseItemLtoS @0x5cf986
// (COutPacket(0x10B) @0x5cf9a4). nTI/nPos args NOT written. Encode1(8) @0x5cf9b2,
// Encode4 nITCSN @0x5cf9c3.
func TestItcOperationMoveLtoSByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationMoveLtoS(0x08, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x08)            // Encode1(8u) mode byte @0x5cf9b2
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5cf9c3
	if !bytes.Equal(got, want) {
		t.Fatalf("MoveLtoS (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationPlaceBid version=gms_v87 ida=0x5f45b1
//
// ITC_OPERATION mode 0x13 place-bid. Send inlined into
// CITCBidAuctionDlg::OnButtonClicked @0x5f45b1 (a2==1 confirm-bid branch,
// COutPacket(0x10B) @0x5f4752). Encode order @0x5f475f..0x5f478c:
//
//	Encode1(0x13u)            @0x5f475f  mode byte
//	Encode4(*(this[51]+32))   @0x5f4770  nITCSN
//	Encode4(this[44])         @0x5f477e  bidPrice
//	Encode4(this[43])         @0x5f478c  bidRange
func TestItcOperationPlaceBidByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationPlaceBid(0x13, 123456, 5000, 100)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x13)            // Encode1(0x13u) mode byte @0x5f475f
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5f4770
	want = append(want, le32(5000)...)   // Encode4 bidPrice @0x5f477e
	want = append(want, le32(100)...)    // Encode4 bidRange @0x5f478c
	if !bytes.Equal(got, want) {
		t.Fatalf("PlaceBid (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSetZzim version=gms_v87 ida=0x5cf600
//
// ITC_OPERATION mode 0x09 set-zzim. CITC::OnSetZzim @0x5cf600
// (COutPacket(267) @0x5cf61e). Encode1(9) @0x5cf62c, Encode4 nITCSN @0x5cf63d.
func TestItcOperationSetZzimByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationSetZzim(0x09, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x09)            // Encode1(9u) mode byte @0x5cf62c
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5cf63d
	if !bytes.Equal(got, want) {
		t.Fatalf("SetZzim (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyZzim version=gms_v87 ida=0x5cf673
//
// ITC_OPERATION mode 0x11 buy-zzim. CITC::OnBuyZzim @0x5cf673 (YesNo gate;
// COutPacket(267) @0x5cf6bc). Encode1(0x11) @0x5cf6c9, Encode4 nITCSN @0x5cf6da.
func TestItcOperationBuyZzimByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationBuyZzim(0x11, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x11)            // Encode1(0x11u) mode byte @0x5cf6c9
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5cf6da
	if !bytes.Equal(got, want) {
		t.Fatalf("BuyZzim (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationDeleteZzim version=gms_v87 ida=0x5cf711
//
// ITC_OPERATION mode 0x0A delete-zzim. CITC::OnDeleteZzim @0x5cf711
// (COutPacket(267) @0x5cf72f). Encode1(0xA) @0x5cf73d, Encode4 nITCSN @0x5cf74e.
func TestItcOperationDeleteZzimByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationDeleteZzim(0x0A, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0A)            // Encode1(0xAu) mode byte @0x5cf73d
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5cf74e
	if !bytes.Equal(got, want) {
		t.Fatalf("DeleteZzim (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationViewWish version=gms_v87 ida=0x5cf784
//
// ITC_OPERATION mode 0x0B view-wish. CITC::OnViewWish @0x5cf784
// (COutPacket(267) @0x5cf7a2). Encode1(0xB) @0x5cf7b0, Encode4 nITCSN @0x5cf7c1.
func TestItcOperationViewWishByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationViewWish(0x0B, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0B)            // Encode1(0xBu) mode byte @0x5cf7b0
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5cf7c1
	if !bytes.Equal(got, want) {
		t.Fatalf("ViewWish (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyWish version=gms_v87 ida=0x5cf7f7
//
// ITC_OPERATION mode 0x0C buy-wish. CITC::OnBuyWish @0x5cf7f7
// (COutPacket(267) @0x5cf815). Encode1(0xC) @0x5cf823, Encode4 nITCSN @0x5cf834.
func TestItcOperationBuyWishByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationBuyWish(0x0C, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0C)            // Encode1(0xCu) mode byte @0x5cf823
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5cf834
	if !bytes.Equal(got, want) {
		t.Fatalf("BuyWish (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelWish version=gms_v87 ida=0x5cf86a
//
// ITC_OPERATION mode 0x0D cancel-wish. CITC::OnCancelWish @0x5cf86a
// (COutPacket(267) @0x5cf888). Encode1(0xD) @0x5cf896, Encode4 nITCSN @0x5cf8a7.
func TestItcOperationCancelWishByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationCancelWish(0x0D, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0D)            // Encode1(0xDu) mode byte @0x5cf896
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5cf8a7
	if !bytes.Equal(got, want) {
		t.Fatalf("CancelWish (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterWishEntry version=gms_v87 ida=0x5cf2a5
//
// ITC_OPERATION mode 0x04 register-wish-entry. CITC::OnRegisterWishEntry
// @0x5cf2a5 (110-NX floor guard, not on wire; COutPacket(0x10B) @0x5cf31f).
// Encode order @0x5cf32f..0x5cf392:
//
//	Encode1(4u)                      @0x5cf32f  mode byte
//	Encode4(*(this+8152))            @0x5cf33d  itemId
//	Encode4(*(this+8156))            @0x5cf34b  price
//	Encode4(*(this+8160))            @0x5cf359  count
//	Encode1(*(this+8165))            @0x5cf36a  duration
//	Encode1(*(this+8164))            @0x5cf37b  feeOption
//	EncodeStr(this+8168)             @0x5cf392  description (uint16 len + bytes)
func TestItcOperationRegisterWishEntryByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationRegisterWishEntry(0x04, 2000000, 1000, 5, 0x07, 0x01, "wish")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x04)              // Encode1(4u) mode byte @0x5cf32f
	want = append(want, le32(2000000)...)  // Encode4 itemId @0x5cf33d
	want = append(want, le32(1000)...)     // Encode4 price @0x5cf34b
	want = append(want, le32(5)...)        // Encode4 count @0x5cf359
	want = append(want, 0x07)              // Encode1 duration @0x5cf36a
	want = append(want, 0x01)              // Encode1 feeOption @0x5cf37b
	want = append(want, 0x04, 0x00)        // EncodeStr len prefix (uint16 LE = 4) @0x5cf392
	want = append(want, []byte("wish")...) // EncodeStr bytes @0x5cf392
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterWishEntry (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategory version=gms_v87 ida=0x5ceffa
//
// ITC_OPERATION mode 0x05 change-browse-category. CITC::OnChangedCategory
// @0x5ceffa (COutPacket(0x10B) @0x5cf042). Only nCategory variable; rest const.
// Encode order @0x5cf04f..0x5cf0a2:
//
//	Encode1(5u)        @0x5cf04f  mode byte
//	Encode4(a2)        @0x5cf05a  category
//	Encode4(0)         @0x5cf063  categorySub (const 0)
//	Encode4(0)         @0x5cf06c  page (const 0)
//	Encode1(1u)        @0x5cf078  sortType (const 1)
//	Encode1(1u)        @0x5cf081  sortColumn (const 1)
//	Encode4(1u)        @0x5cf08a  searchOption (const 1)
//	EncodeStr("")      @0x5cf0a2  searchCondition (const empty)
func TestItcOperationChangedCategoryByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationChangedCategory(0x05, 3)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x05)       // Encode1(5u) mode byte @0x5cf04f
	want = append(want, le32(3)...) // Encode4 category @0x5cf05a
	want = append(want, le32(0)...) // Encode4 categorySub @0x5cf063
	want = append(want, le32(0)...) // Encode4 page @0x5cf06c
	want = append(want, 0x01)       // Encode1 sortType @0x5cf078
	want = append(want, 0x01)       // Encode1 sortColumn @0x5cf081
	want = append(want, le32(1)...) // Encode4 searchOption @0x5cf08a
	want = append(want, 0x00, 0x00) // EncodeStr("") uint16 len 0 @0x5cf0a2
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedCategory (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategorySub version=gms_v87 ida=0x5cf0d9
//
// ITC_OPERATION mode 0x05 change-sub-category. CITC::OnChangedCategorySub
// @0x5cf0d9 (COutPacket(267) @0x5cf0fe). page=0 const. Encode order
// @0x5cf10b..0x5cf190:
//
//	Encode1(5u)            @0x5cf10b  mode byte
//	Encode4(*(this+104))   @0x5cf116  category
//	Encode4(a2)            @0x5cf122  categorySub
//	Encode4(0)             @0x5cf12b  page (const 0)
//	Encode1(arg4)          @0x5cf136  sortType
//	Encode1(a4)            @0x5cf141  sortColumn
//	Encode4(searchOption)  @0x5cf174/@0x5cf150  searchOption
//	EncodeStr(cond)        @0x5cf190  searchCondition
func TestItcOperationChangedCategorySubByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationChangedCategorySub(0x05, 3, 2, 0x01, 0x01, 1, "")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x05)       // Encode1(5u) mode byte @0x5cf10b
	want = append(want, le32(3)...) // Encode4 category @0x5cf116
	want = append(want, le32(2)...) // Encode4 categorySub @0x5cf122
	want = append(want, le32(0)...) // Encode4 page @0x5cf12b
	want = append(want, 0x01)       // Encode1 sortType @0x5cf136
	want = append(want, 0x01)       // Encode1 sortColumn @0x5cf141
	want = append(want, le32(1)...) // Encode4 searchOption @0x5cf150
	want = append(want, 0x00, 0x00) // EncodeStr("") uint16 len 0 @0x5cf190
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedCategorySub (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedPage version=gms_v87 ida=0x5cf1c8
//
// ITC_OPERATION mode 0x05 change-browse-page. CITC::OnChangedPage @0x5cf1c8
// (COutPacket(267) @0x5cf1ea). All 8 fields from CITC state. Encode order
// @0x5cf1f8..0x5cf260:
//
//	Encode1(5u)            @0x5cf1f8  mode byte
//	Encode4(*(this+104))   @0x5cf203  category
//	Encode4(*(this+108))   @0x5cf20e  categorySub
//	Encode4(a2)            @0x5cf219  page
//	Encode1(*(this+96))    @0x5cf227  sortType
//	Encode1(*(this+100))   @0x5cf235  sortColumn
//	Encode4(*(this+8204))  @0x5cf243  searchOption
//	EncodeStr(this+8208)   @0x5cf260  searchCondition
func TestItcOperationChangedPageByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationChangedPage(0x05, 3, 2, 4, 0x01, 0x01, 1, "abc")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x05)             // Encode1(5u) mode byte @0x5cf1f8
	want = append(want, le32(3)...)       // Encode4 category @0x5cf203
	want = append(want, le32(2)...)       // Encode4 categorySub @0x5cf20e
	want = append(want, le32(4)...)       // Encode4 page @0x5cf219
	want = append(want, 0x01)             // Encode1 sortType @0x5cf227
	want = append(want, 0x01)             // Encode1 sortColumn @0x5cf235
	want = append(want, le32(1)...)       // Encode4 searchOption @0x5cf243
	want = append(want, 0x03, 0x00)       // EncodeStr len prefix (uint16 LE = 3) @0x5cf260
	want = append(want, []byte("abc")...) // EncodeStr bytes @0x5cf260
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedPage (v87):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationTabSearch version=gms_v87 ida=0x5e7a64
//
// ITC_OPERATION mode 0x06 tab/search-by-name. Send inlined into
// CITCWnd_Tab::OnButtonClicked @0x5e7a64 (a2==1004 search button, COutPacket(0x10B)
// @0x5e7b03/@0x5e7bcc). 6-field body, NO sort bytes. Encode order
// (advanced-search branch @0x5e7bda; character-name branch @0x5e7b11 is identical):
//
//	Encode1(6u)            @0x5e7bda  mode byte
//	Encode4(m_nSelect+1)   @0x5e7be3  category
//	Encode4(v21)           @0x5e7bee  categorySub
//	Encode4(0)             @0x5e7bf7  page (const 0)
//	Encode4(searchOption)  @0x5e7c02  searchOption
//	EncodeStr(searchName)  @0x5e7c1b  searchCondition
func TestItcOperationTabSearchByteOutput_v87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewItcOperationTabSearch(0x06, 3, 2, 0, "hero")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x06)              // Encode1(6u) mode byte @0x5e7bda
	want = append(want, le32(3)...)        // Encode4 category @0x5e7be3
	want = append(want, le32(2)...)        // Encode4 categorySub @0x5e7bee
	want = append(want, le32(0)...)        // Encode4 page @0x5e7bf7
	want = append(want, le32(0)...)        // Encode4 searchOption @0x5e7c02
	want = append(want, 0x04, 0x00)        // EncodeStr len prefix (uint16 LE = 4) @0x5e7c1b
	want = append(want, []byte("hero")...) // EncodeStr bytes @0x5e7c1b
	if !bytes.Equal(got, want) {
		t.Fatalf("TabSearch (v87):\n got %v\nwant %v", got, want)
	}
}

// =============================================================================
// gms_v84 — all 18 ITC_OPERATION serverbound arms, dispatcher opcode 0x104/260
// (GMS_v84.1_U_DEVM.exe, IDA port 13337). v84 is a DEVM build, but the CITC
// sender methods were NOT symbol-named (unlike v87/v95); they decompile as a
// dense sub_5AEF* cluster right after the named CITC::OnStatusCharge @0x5aef76
// and TrySendQueryCashRequest @0x5af26a siblings, and were named in the IDB by
// reading their COutPacket opcode + Encode order (PlaceBid/TabSearch inline into
// the bid-dialog/tab-button handlers, same as v83/v87/v95). The dispatcher
// opcode COutPacket(260) is confirmed at CITC::OnRegisterSaleEntry @0x5aefff —
// a +7 shift from the registry's stale v83-carryover 253 (corrected to 260 in
// the preceding fix commit). EVERY arm's mode byte is BYTE-IDENTICAL to gms_v95
// and gms_v87 — no divergence. The item-slot blob is written by sub_4EA6F8
// (v84's GW_ItemSlotBase writer; v83 sub_4E33D8, v87 sub_502670, v95
// GW_ItemSlotBase::Encode) — same model.Asset contract. Per the dispatcher-
// family rule each arm gets its OWN byte fixture.
// =============================================================================

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterSale version=gms_v84 ida=0x5aefd2
//
// ITC_OPERATION mode 2 register-fixed-price. Derived from
// CITC::OnRegisterSaleEntry @0x5aefd2 (COutPacket(260) @0x5aefff), arg0==0
// branch. Encode order @0x5af12e..0x5af171:
//
//	Encode1(2u)          @0x5af12e  mode byte
//	sub_4EA6F8(&pkt)     @0x5af13a  item-slot blob (model.Asset)
//	Encode4(a4)          @0x5af145  quantity
//	Encode4(v20)         @0x5af150  commodityId
//	Encode4(a3)          @0x5af15b  price
//	Encode1(v21)         @0x5af166  type
//	Encode1(v19)         @0x5af171  flag
func TestItcOperationRegisterSaleByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationRegisterSale(0x02, asset, 5, 2000000, 1000, 0x01, 0x00)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x02)             // Encode1(2u) mode byte @0x5af12e
	want = append(want, assetBytes...)    // sub_4EA6F8 item-slot blob @0x5af13a
	want = append(want, le32(5)...)       // Encode4 quantity @0x5af145
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x5af150
	want = append(want, le32(1000)...)    // Encode4 price @0x5af15b
	want = append(want, 0x01)             // Encode1 type @0x5af166
	want = append(want, 0x00)             // Encode1 flag @0x5af171
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterSale (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterAuction version=gms_v84 ida=0x5af045
//
// ITC_OPERATION mode 0x12 register-auction. Derived from
// CITC::OnRegisterSaleEntry @0x5aefd2 (COutPacket(260) @0x5aefff), arg0==1
// branch. Encode order @0x5af064..0x5af0bd:
//
//	Encode1(0x12u)       @0x5af064  mode byte
//	sub_4EA6F8(&pkt)     @0x5af070  item-slot blob (model.Asset)
//	Encode4(a4)          @0x5af07b  quantity
//	Encode4(v20)         @0x5af086  commodityId
//	Encode4(a2)          @0x5af091  selector (==1)
//	Encode4(v18)         @0x5af09c  buyNowPrice
//	Encode1(v21)         @0x5af0a7  type
//	Encode1(v19)         @0x5af0b2  flag
//	Encode4(v17)         @0x5af0bd  durationHrs
func TestItcOperationRegisterAuctionByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationRegisterAuction(0x12, asset, 5, 2000000, 1, 5000, 0x01, 0x00, 24)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x12)             // Encode1(0x12u) mode byte @0x5af064
	want = append(want, assetBytes...)    // sub_4EA6F8 item-slot blob @0x5af070
	want = append(want, le32(5)...)       // Encode4 quantity @0x5af07b
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x5af086
	want = append(want, le32(1)...)       // Encode4 selector @0x5af091
	want = append(want, le32(5000)...)    // Encode4 buyNowPrice @0x5af09c
	want = append(want, 0x01)             // Encode1 type @0x5af0a7
	want = append(want, 0x00)             // Encode1 flag @0x5af0b2
	want = append(want, le32(24)...)      // Encode4 durationHrs @0x5af0bd
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterAuction (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSaleCurrentItem version=gms_v84 ida=0x5af1db
//
// ITC_OPERATION mode 3 sell-currently-selected-item. Derived from
// CITC::OnSaleCurrentItem @0x5af1db (COutPacket(260) @0x5af1f9). Encode order
// @0x5af207..0x5af234:
//
//	Encode1(3u)          @0x5af207  mode byte
//	Encode1(a2)          @0x5af212  type
//	Encode4(a3)          @0x5af21d  slotPos
//	sub_4EA6F8(&pkt)     @0x5af229  item-slot blob (model.Asset)
//	Encode4(a5)          @0x5af234  commodityId
func TestItcOperationSaleCurrentItemByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)

	input := NewItcOperationSaleCurrentItem(0x03, 0x01, 7, asset, 2000000)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x03)             // Encode1(3u) mode byte @0x5af207
	want = append(want, 0x01)             // Encode1 type @0x5af212
	want = append(want, le32(7)...)       // Encode4 slotPos @0x5af21d
	want = append(want, assetBytes...)    // sub_4EA6F8 item-slot blob @0x5af229
	want = append(want, le32(2000000)...) // Encode4 commodityId @0x5af234
	if !bytes.Equal(got, want) {
		t.Fatalf("SaleCurrentItem (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuy version=gms_v84 ida=0x5af9fa
//
// ITC_OPERATION mode 0x10 buy-fixed-price. CITC::OnBuy @0x5af9fa
// (COutPacket(260) @0x5afa18). Encode1(0x10) @0x5afa26, Encode4 nITCSN @0x5afa37.
func TestItcOperationBuyByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationBuy(0x10, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x10)            // Encode1(0x10u) mode byte @0x5afa26
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5afa37
	if !bytes.Equal(got, want) {
		t.Fatalf("Buy (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyAuctionImm version=gms_v84 ida=0x5afa6d
//
// ITC_OPERATION mode 0x14 buy-now-on-auction. CITC::OnBuyAuctionImm @0x5afa6d
// (COutPacket(260) @0x5afa8b). Encode1(0x14) @0x5afa99, Encode4 nITCSN @0x5afaaa.
func TestItcOperationBuyAuctionImmByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationBuyAuctionImm(0x14, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x14)            // Encode1(0x14u) mode byte @0x5afa99
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5afaaa
	if !bytes.Equal(got, want) {
		t.Fatalf("BuyAuctionImm (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelSale version=gms_v84 ida=0x5afedd
//
// ITC_OPERATION mode 0x07 cancel-sale. CITC::OnCancelSaleItem @0x5afedd
// (YesNo gate sub_9D3ABC==6; COutPacket(260) @0x5aff33). Encode1(7) @0x5aff40,
// Encode4 nITCSN @0x5aff4e.
func TestItcOperationCancelSaleByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationCancelSale(0x07, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x07)            // Encode1(7u) mode byte @0x5aff40
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5aff4e
	if !bytes.Equal(got, want) {
		t.Fatalf("CancelSale (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationMoveLtoS version=gms_v84 ida=0x5aff86
//
// ITC_OPERATION mode 0x08 take-home. CITC::OnMoveITCPurchaseItemLtoS @0x5aff86
// (COutPacket(260) @0x5affa4). nTI/nPos args NOT written. Encode1(8) @0x5affb2,
// Encode4 nITCSN @0x5affc3.
func TestItcOperationMoveLtoSByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationMoveLtoS(0x08, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x08)            // Encode1(8u) mode byte @0x5affb2
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5affc3
	if !bytes.Equal(got, want) {
		t.Fatalf("MoveLtoS (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationPlaceBid version=gms_v84 ida=0x5d3ec7
//
// ITC_OPERATION mode 0x13 place-bid. Send inlined into
// CITCBidAuctionDlg::OnButtonClicked @0x5d3ec7 (a2==1 confirm-bid branch,
// COutPacket(260) @0x5d4068). Encode order @0x5d4075..0x5d40a2:
//
//	Encode1(0x13u)            @0x5d4075  mode byte
//	Encode4(*(this[47]+32))   @0x5d4086  nITCSN
//	Encode4(this[40])         @0x5d4094  bidPrice
//	Encode4(this[39])         @0x5d40a2  bidRange
func TestItcOperationPlaceBidByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationPlaceBid(0x13, 123456, 5000, 100)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x13)            // Encode1(0x13u) mode byte @0x5d4075
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5d4086
	want = append(want, le32(5000)...)   // Encode4 bidPrice @0x5d4094
	want = append(want, le32(100)...)    // Encode4 bidRange @0x5d40a2
	if !bytes.Equal(got, want) {
		t.Fatalf("PlaceBid (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSetZzim version=gms_v84 ida=0x5afc00
//
// ITC_OPERATION mode 0x09 set-zzim. CITC::OnSetZzim @0x5afc00
// (COutPacket(260) @0x5afc1e). Encode1(9) @0x5afc2c, Encode4 nITCSN @0x5afc3d.
func TestItcOperationSetZzimByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationSetZzim(0x09, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x09)            // Encode1(9u) mode byte @0x5afc2c
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5afc3d
	if !bytes.Equal(got, want) {
		t.Fatalf("SetZzim (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyZzim version=gms_v84 ida=0x5afc73
//
// ITC_OPERATION mode 0x11 buy-zzim. CITC::OnBuyZzim @0x5afc73 (YesNo gate
// sub_9D3ABC==6; COutPacket(260) @0x5afcbc). Encode1(0x11) @0x5afcc9,
// Encode4 nITCSN @0x5afcda.
func TestItcOperationBuyZzimByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationBuyZzim(0x11, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x11)            // Encode1(0x11u) mode byte @0x5afcc9
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5afcda
	if !bytes.Equal(got, want) {
		t.Fatalf("BuyZzim (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationDeleteZzim version=gms_v84 ida=0x5afd11
//
// ITC_OPERATION mode 0x0A delete-zzim. CITC::OnDeleteZzim @0x5afd11
// (COutPacket(260) @0x5afd2f). Encode1(0xA) @0x5afd3d, Encode4 nITCSN @0x5afd4e.
func TestItcOperationDeleteZzimByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationDeleteZzim(0x0A, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0A)            // Encode1(0xAu) mode byte @0x5afd3d
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5afd4e
	if !bytes.Equal(got, want) {
		t.Fatalf("DeleteZzim (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationViewWish version=gms_v84 ida=0x5afd84
//
// ITC_OPERATION mode 0x0B view-wish. CITC::OnViewWish @0x5afd84
// (COutPacket(260) @0x5afda2). Encode1(0xB) @0x5afdb0, Encode4 nITCSN @0x5afdc1.
func TestItcOperationViewWishByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationViewWish(0x0B, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0B)            // Encode1(0xBu) mode byte @0x5afdb0
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5afdc1
	if !bytes.Equal(got, want) {
		t.Fatalf("ViewWish (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyWish version=gms_v84 ida=0x5afdf7
//
// ITC_OPERATION mode 0x0C buy-wish. CITC::OnBuyWish @0x5afdf7
// (COutPacket(260) @0x5afe15). Encode1(0xC) @0x5afe23, Encode4 nITCSN @0x5afe34.
func TestItcOperationBuyWishByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationBuyWish(0x0C, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0C)            // Encode1(0xCu) mode byte @0x5afe23
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5afe34
	if !bytes.Equal(got, want) {
		t.Fatalf("BuyWish (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelWish version=gms_v84 ida=0x5afe6a
//
// ITC_OPERATION mode 0x0D cancel-wish. CITC::OnCancelWish @0x5afe6a
// (COutPacket(260) @0x5afe88). Encode1(0xD) @0x5afe96, Encode4 nITCSN @0x5afea7.
func TestItcOperationCancelWishByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationCancelWish(0x0D, 123456)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x0D)            // Encode1(0xDu) mode byte @0x5afe96
	want = append(want, le32(123456)...) // Encode4 nITCSN @0x5afea7
	if !bytes.Equal(got, want) {
		t.Fatalf("CancelWish (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterWishEntry version=gms_v84 ida=0x5af8a5
//
// ITC_OPERATION mode 0x04 register-wish-entry. CITC::OnRegisterWishEntry
// @0x5af8a5 (110-NX floor guard a3<=109 @0x5af8d6, not on wire; COutPacket(260)
// @0x5af91f). Encode order @0x5af92f..0x5af992:
//
//	Encode1(4u)                      @0x5af92f  mode byte
//	Encode4(*(this+7864))            @0x5af93d  itemId
//	Encode4(*(this+7868))            @0x5af94b  price
//	Encode4(*(this+7872))            @0x5af959  count
//	Encode1(*(this+7877))            @0x5af96a  duration
//	Encode1(*(this+7876))            @0x5af97b  feeOption
//	EncodeStr(this+7880)             @0x5af992  description (uint16 len + bytes)
func TestItcOperationRegisterWishEntryByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationRegisterWishEntry(0x04, 2000000, 1000, 5, 0x07, 0x01, "wish")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x04)              // Encode1(4u) mode byte @0x5af92f
	want = append(want, le32(2000000)...)  // Encode4 itemId @0x5af93d
	want = append(want, le32(1000)...)     // Encode4 price @0x5af94b
	want = append(want, le32(5)...)        // Encode4 count @0x5af959
	want = append(want, 0x07)              // Encode1 duration @0x5af96a
	want = append(want, 0x01)              // Encode1 feeOption @0x5af97b
	want = append(want, 0x04, 0x00)        // EncodeStr len prefix (uint16 LE = 4) @0x5af992
	want = append(want, []byte("wish")...) // EncodeStr bytes @0x5af992
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterWishEntry (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategory version=gms_v84 ida=0x5af5fa
//
// ITC_OPERATION mode 0x05 change-browse-category. CITC::OnChangedCategory
// @0x5af5fa (COutPacket(260) @0x5af642). Only nCategory variable; rest const.
// Encode order @0x5af64f..0x5af6a2:
//
//	Encode1(5u)        @0x5af64f  mode byte
//	Encode4(a2)        @0x5af65a  category
//	Encode4(0)         @0x5af663  categorySub (const 0)
//	Encode4(0)         @0x5af66c  page (const 0)
//	Encode1(1u)        @0x5af678  sortType (const 1)
//	Encode1(1u)        @0x5af681  sortColumn (const 1)
//	Encode4(1u)        @0x5af68a  searchOption (const 1)
//	EncodeStr("")      @0x5af6a2  searchCondition (const empty)
func TestItcOperationChangedCategoryByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationChangedCategory(0x05, 3)
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x05)       // Encode1(5u) mode byte @0x5af64f
	want = append(want, le32(3)...) // Encode4 category @0x5af65a
	want = append(want, le32(0)...) // Encode4 categorySub @0x5af663
	want = append(want, le32(0)...) // Encode4 page @0x5af66c
	want = append(want, 0x01)       // Encode1 sortType @0x5af678
	want = append(want, 0x01)       // Encode1 sortColumn @0x5af681
	want = append(want, le32(1)...) // Encode4 searchOption @0x5af68a
	want = append(want, 0x00, 0x00) // EncodeStr("") uint16 len 0 @0x5af6a2
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedCategory (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategorySub version=gms_v84 ida=0x5af6d9
//
// ITC_OPERATION mode 0x05 change-sub-category. CITC::OnChangedCategorySub
// @0x5af6d9 (COutPacket(260) @0x5af6fe). page=0 const. Encode order
// @0x5af70b..0x5af790:
//
//	Encode1(5u)            @0x5af70b  mode byte
//	Encode4(*(this+104))   @0x5af716  category
//	Encode4(a2)            @0x5af722  categorySub
//	Encode4(0)             @0x5af72b  page (const 0)
//	Encode1(a3)            @0x5af736  sortType
//	Encode1(a4)            @0x5af741  sortColumn
//	Encode4(searchOption)  @0x5af750  searchOption (else-branch const 1)
//	EncodeStr(cond)        @0x5af790  searchCondition
func TestItcOperationChangedCategorySubByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationChangedCategorySub(0x05, 3, 2, 0x01, 0x01, 1, "")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x05)       // Encode1(5u) mode byte @0x5af70b
	want = append(want, le32(3)...) // Encode4 category @0x5af716
	want = append(want, le32(2)...) // Encode4 categorySub @0x5af722
	want = append(want, le32(0)...) // Encode4 page @0x5af72b
	want = append(want, 0x01)       // Encode1 sortType @0x5af736
	want = append(want, 0x01)       // Encode1 sortColumn @0x5af741
	want = append(want, le32(1)...) // Encode4 searchOption @0x5af750
	want = append(want, 0x00, 0x00) // EncodeStr("") uint16 len 0 @0x5af790
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedCategorySub (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedPage version=gms_v84 ida=0x5af7c8
//
// ITC_OPERATION mode 0x05 change-browse-page. CITC::OnChangedPage @0x5af7c8
// (COutPacket(260) @0x5af7ea). All 8 fields from CITC state. Encode order
// @0x5af7f8..0x5af860:
//
//	Encode1(5u)            @0x5af7f8  mode byte
//	Encode4(*(this+104))   @0x5af803  category
//	Encode4(*(this+108))   @0x5af80e  categorySub
//	Encode4(a2)            @0x5af819  page
//	Encode1(*(this+96))    @0x5af827  sortType
//	Encode1(*(this+100))   @0x5af835  sortColumn
//	Encode4(*(this+7916))  @0x5af843  searchOption
//	EncodeStr(this+7920)   @0x5af860  searchCondition
func TestItcOperationChangedPageByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationChangedPage(0x05, 3, 2, 4, 0x01, 0x01, 1, "abc")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x05)             // Encode1(5u) mode byte @0x5af7f8
	want = append(want, le32(3)...)       // Encode4 category @0x5af803
	want = append(want, le32(2)...)       // Encode4 categorySub @0x5af80e
	want = append(want, le32(4)...)       // Encode4 page @0x5af819
	want = append(want, 0x01)             // Encode1 sortType @0x5af827
	want = append(want, 0x01)             // Encode1 sortColumn @0x5af835
	want = append(want, le32(1)...)       // Encode4 searchOption @0x5af843
	want = append(want, 0x03, 0x00)       // EncodeStr len prefix (uint16 LE = 3) @0x5af860
	want = append(want, []byte("abc")...) // EncodeStr bytes @0x5af860
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedPage (v84):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationTabSearch version=gms_v84 ida=0x5c77ca
//
// ITC_OPERATION mode 0x06 tab/search-by-name. Send inlined into
// CITCWnd_Tab::OnButtonClicked @0x5c77ca (a2==1004 search button, COutPacket(260)
// @0x5c7869/@0x5c7932). 6-field body, NO sort bytes. Encode order
// (advanced-search branch @0x5c7940; character-name branch @0x5c7877 is identical):
//
//	Encode1(6u)            @0x5c7940  mode byte
//	Encode4(v3=m_nSelect+1)@0x5c7949  category
//	Encode4(v17)           @0x5c7954  categorySub
//	Encode4(0)             @0x5c795d  page (const 0)
//	Encode4(v16)           @0x5c7968  searchOption
//	EncodeStr(name)        @0x5c7981  searchCondition
func TestItcOperationTabSearchByteOutput_v84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewItcOperationTabSearch(0x06, 3, 2, 0, "hero")
	got := pt.Encode(t, ctx, input.Encode, nil)

	var want []byte
	want = append(want, 0x06)              // Encode1(6u) mode byte @0x5c7940
	want = append(want, le32(3)...)        // Encode4 category @0x5c7949
	want = append(want, le32(2)...)        // Encode4 categorySub @0x5c7954
	want = append(want, le32(0)...)        // Encode4 page @0x5c795d
	want = append(want, le32(0)...)        // Encode4 searchOption @0x5c7968
	want = append(want, 0x04, 0x00)        // EncodeStr len prefix (uint16 LE = 4) @0x5c7981
	want = append(want, []byte("hero")...) // EncodeStr bytes @0x5c7981
	if !bytes.Equal(got, want) {
		t.Fatalf("TabSearch (v84):\n got %v\nwant %v", got, want)
	}
}
