package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v79 MTS/ITC (CITC) serverbound byte fixtures (task-113 v79 MTS port,
// GMS_v79_1_DEVM.exe, IDA port 13340). The codecs are version-agnostic; the v79
// wire bodies are byte-identical to gms_v83 and every mode byte was re-verified
// from the v79 CITC send switch. Only the container opcodes differ:
// ENTER_MTS 0x99/153, ITC_STATUS_CHARGE 0xF1/241, ITC_QUERY_CASH_REQUEST
// 0xF2/242, ITC_OPERATION 0xF3/243. itcTestAsset/le32 are defined in
// itc_operation_test.go (same package).

// packet-audit:verify packet=field/serverbound/FieldEnterMts version=gms_v79 ida=0x95dd85
//
// ENTER_MTS (gms_v79 opcode 0x99/153). CWvsContext::SendMigrateToITCRequest
// @0x95dd85; send site COutPacket(153) @0x95de9e then SendPacket with ZERO
// Encode calls — bodiless. Guest/lie-detector/map-flag guards emit chat/notice
// and return early; none writes the wire.
func TestEnterMtsByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := EnterMts{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("EnterMts body (v79): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcStatusCharge version=gms_v79 ida=0x57a1b0
//
// ITC_STATUS_CHARGE (gms_v79 opcode 0xF1/241). CITC::OnStatusCharge @0x57a1b0;
// m_bITCRequestSent latch guards COutPacket(241) @0x57a1d2 + immediate
// SendPacket — bodiless.
func TestItcStatusChargeByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ItcStatusCharge{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcStatusCharge body (v79): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcQueryCashRequest version=gms_v79 ida=0x57a4a4
//
// ITC_QUERY_CASH_REQUEST (gms_v79 opcode 0xF2/242). CITC::TrySendQueryCashRequest
// @0x57a4a4; latch guards COutPacket(242) @0x57a4c6 + immediate SendPacket —
// bodiless.
func TestItcQueryCashRequestByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := ItcQueryCashRequest{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcQueryCashRequest body (v79): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// ITC_OPERATION arms (gms_v79 dispatcher opcode 0xF3/243). Each arm's mode byte
// was verified from the v79 sender; bodies are byte-identical to gms_v83.

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterSale version=gms_v79 ida=0x57a20c
//
// mode 2 register-fixed-price. CITC::OnRegisterSaleEntry @0x57a20c arg0==0 branch:
// Encode1(2), item blob, Encode4 slotPos/quantity/price, Encode1 type/flag.
func TestItcOperationRegisterSaleByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)
	got := pt.Encode(t, ctx, NewItcOperationRegisterSale(0x02, asset, 5, 2000000, 1000, 0x01, 0x00).Encode, nil)
	var want []byte
	want = append(want, 0x02)
	want = append(want, assetBytes...)
	want = append(want, le32(5)...)
	want = append(want, le32(2000000)...)
	want = append(want, le32(1000)...)
	want = append(want, 0x01, 0x00)
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterSale (v79):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterAuction version=gms_v79 ida=0x57a20c
//
// mode 0x12 register-auction. CITC::OnRegisterSaleEntry @0x57a20c arg0==1 branch:
// Encode1(0x12), item blob, Encode4 slotPos/quantity/selector/buyNow, Encode1
// duration/flag, Encode4 increment.
func TestItcOperationRegisterAuctionByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)
	got := pt.Encode(t, ctx, NewItcOperationRegisterAuction(0x12, asset, 5, 2000000, 1, 5000, 0x01, 0x00, 24).Encode, nil)
	var want []byte
	want = append(want, 0x12)
	want = append(want, assetBytes...)
	want = append(want, le32(5)...)
	want = append(want, le32(2000000)...)
	want = append(want, le32(1)...)
	want = append(want, le32(5000)...)
	want = append(want, 0x01, 0x00)
	want = append(want, le32(24)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterAuction (v79):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSaleCurrentItem version=gms_v79 ida=0x57a415
//
// mode 3 want-ad offer. CITC::OnSaleCurrentItem @0x57a415: Encode1(3), Encode1
// type, Encode4 slotPos, item blob, Encode4 wishSerial.
func TestItcOperationSaleCurrentItemByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)
	got := pt.Encode(t, ctx, NewItcOperationSaleCurrentItem(0x03, 0x01, 7, asset, 2000000).Encode, nil)
	var want []byte
	want = append(want, 0x03, 0x01)
	want = append(want, le32(7)...)
	want = append(want, assetBytes...)
	want = append(want, le32(2000000)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("SaleCurrentItem (v79):\n got %v\nwant %v", got, want)
	}
}

// serialOnlyV79 asserts a mode+Encode4(nITCSN) arm for gms_v79.
func serialOnlyV79(t *testing.T, mode byte, got []byte) {
	t.Helper()
	var want []byte
	want = append(want, mode)
	want = append(want, le32(123456)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("serial-only mode 0x%02x (v79):\n got %v\nwant %v", mode, got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuy version=gms_v79 ida=0x57ac34
func TestItcOperationBuyByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	serialOnlyV79(t, 0x10, pt.Encode(t, ctx, NewItcOperationBuy(0x10, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyAuctionImm version=gms_v79 ida=0x57aca7
func TestItcOperationBuyAuctionImmByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	serialOnlyV79(t, 0x14, pt.Encode(t, ctx, NewItcOperationBuyAuctionImm(0x14, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelSale version=gms_v79 ida=0x57b117
func TestItcOperationCancelSaleByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	serialOnlyV79(t, 0x07, pt.Encode(t, ctx, NewItcOperationCancelSale(0x07, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationMoveLtoS version=gms_v79 ida=0x57b1c0
func TestItcOperationMoveLtoSByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	serialOnlyV79(t, 0x08, pt.Encode(t, ctx, NewItcOperationMoveLtoS(0x08, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSetZzim version=gms_v79 ida=0x57ae3a
func TestItcOperationSetZzimByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	serialOnlyV79(t, 0x09, pt.Encode(t, ctx, NewItcOperationSetZzim(0x09, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyZzim version=gms_v79 ida=0x57aead
func TestItcOperationBuyZzimByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	serialOnlyV79(t, 0x11, pt.Encode(t, ctx, NewItcOperationBuyZzim(0x11, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationDeleteZzim version=gms_v79 ida=0x57af4b
func TestItcOperationDeleteZzimByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	serialOnlyV79(t, 0x0A, pt.Encode(t, ctx, NewItcOperationDeleteZzim(0x0A, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationViewWish version=gms_v79 ida=0x57afbe
func TestItcOperationViewWishByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	serialOnlyV79(t, 0x0B, pt.Encode(t, ctx, NewItcOperationViewWish(0x0B, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyWish version=gms_v79 ida=0x57b031
func TestItcOperationBuyWishByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	serialOnlyV79(t, 0x0C, pt.Encode(t, ctx, NewItcOperationBuyWish(0x0C, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelWish version=gms_v79 ida=0x57b0a4
func TestItcOperationCancelWishByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	serialOnlyV79(t, 0x0D, pt.Encode(t, ctx, NewItcOperationCancelWish(0x0D, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationPlaceBid version=gms_v79 ida=0x59da55
//
// mode 0x13 place-bid. CITCBidAuctionDlg::OnButtonClicked @0x59da55 (nId==1
// confirm branch): Encode1(0x13), Encode4 nITCSN/bidPrice/bidRange.
func TestItcOperationPlaceBidByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := pt.Encode(t, ctx, NewItcOperationPlaceBid(0x13, 123456, 5000, 100).Encode, nil)
	var want []byte
	want = append(want, 0x13)
	want = append(want, le32(123456)...)
	want = append(want, le32(5000)...)
	want = append(want, le32(100)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("PlaceBid (v79):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterWishEntry version=gms_v79 ida=0x57aadf
//
// mode 4 register-wish-entry. CITC::OnRegisterWishEntry @0x57aadf: Encode1(4),
// Encode4 itemId/price/count, Encode1 duration/feeOption, EncodeStr desc.
func TestItcOperationRegisterWishEntryByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := pt.Encode(t, ctx, NewItcOperationRegisterWishEntry(0x04, 2000000, 1000, 5, 0x07, 0x01, "wish").Encode, nil)
	var want []byte
	want = append(want, 0x04)
	want = append(want, le32(2000000)...)
	want = append(want, le32(1000)...)
	want = append(want, le32(5)...)
	want = append(want, 0x07, 0x01)
	want = append(want, 0x04, 0x00)
	want = append(want, []byte("wish")...)
	if !bytes.Equal(got, want) {
		t.Fatalf("RegisterWishEntry (v79):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategory version=gms_v79 ida=0x57a834
func TestItcOperationChangedCategoryByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := pt.Encode(t, ctx, NewItcOperationChangedCategory(0x05, 3).Encode, nil)
	var want []byte
	want = append(want, 0x05)
	want = append(want, le32(3)...)
	want = append(want, le32(0)...)
	want = append(want, le32(0)...)
	want = append(want, 0x01, 0x01)
	want = append(want, le32(1)...)
	want = append(want, 0x00, 0x00)
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedCategory (v79):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategorySub version=gms_v79 ida=0x57a913
func TestItcOperationChangedCategorySubByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := pt.Encode(t, ctx, NewItcOperationChangedCategorySub(0x05, 3, 2, 0x01, 0x01, 1, "").Encode, nil)
	var want []byte
	want = append(want, 0x05)
	want = append(want, le32(3)...)
	want = append(want, le32(2)...)
	want = append(want, le32(0)...)
	want = append(want, 0x01, 0x01)
	want = append(want, le32(1)...)
	want = append(want, 0x00, 0x00)
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedCategorySub (v79):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedPage version=gms_v79 ida=0x57aa02
func TestItcOperationChangedPageByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := pt.Encode(t, ctx, NewItcOperationChangedPage(0x05, 3, 2, 4, 0x01, 0x01, 1, "abc").Encode, nil)
	var want []byte
	want = append(want, 0x05)
	want = append(want, le32(3)...)
	want = append(want, le32(2)...)
	want = append(want, le32(4)...)
	want = append(want, 0x01, 0x01)
	want = append(want, le32(1)...)
	want = append(want, 0x03, 0x00)
	want = append(want, []byte("abc")...)
	if !bytes.Equal(got, want) {
		t.Fatalf("ChangedPage (v79):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationTabSearch version=gms_v79 ida=0x5919d4
//
// mode 6 tab/search-by-name. CITCWnd_Tab::OnButtonClicked @0x5919d4 (char-name
// branch @0x591b34): Encode1(6), Encode4 category/categorySub/page/searchOption,
// EncodeStr name. 6-field body, NO sort bytes.
func TestItcOperationTabSearchByteOutput_v79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := pt.Encode(t, ctx, NewItcOperationTabSearch(0x06, 3, 2, 0, "hero").Encode, nil)
	var want []byte
	want = append(want, 0x06)
	want = append(want, le32(3)...)
	want = append(want, le32(2)...)
	want = append(want, le32(0)...)
	want = append(want, le32(0)...)
	want = append(want, 0x04, 0x00)
	want = append(want, []byte("hero")...)
	if !bytes.Equal(got, want) {
		t.Fatalf("TabSearch (v79):\n got %v\nwant %v", got, want)
	}
}
