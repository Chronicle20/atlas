package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v61 MTS/ITC (CITC) serverbound byte fixtures (task-113 v61 MTS port,
// GMS_v61.1_U_DEVM.exe, IDA port 13338). The codecs are version-agnostic; the
// v61 wire bodies are byte-identical to gms_v83 and every mode byte was
// re-verified from the v61 CITC send-sites (mostly inlined and renamed in the
// v61 IDB). Only the container opcodes differ: ENTER_MTS 0x87/135,
// ITC_STATUS_CHARGE 0xD5/213, ITC_QUERY_CASH_REQUEST 0xD6/214, ITC_OPERATION
// 0xD7/215. itcTestAsset/le32 are defined in itc_operation_test.go (same
// package).

// packet-audit:verify packet=field/serverbound/FieldEnterMts version=gms_v61 ida=0x839b94
//
// ENTER_MTS (gms_v61 opcode 0x87/135). CWvsContext::SendMigrateToITCRequest
// @0x839b94; send site COutPacket(135) @0x839ca3 then SendPacket with ZERO
// Encode calls — bodiless. Guest/map-flag guards emit chat/notice and return
// early; none writes the wire.
func TestEnterMtsByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := EnterMts{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("EnterMts body (v61): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcStatusCharge version=gms_v61 ida=0x528ed7
//
// ITC_STATUS_CHARGE (gms_v61 opcode 0xD5/213). CITC::OnStatusCharge @0x528ed7
// (inlined, renamed from sub_528ED7; nId 1000 charge button); latch set before
// COutPacket(213) + immediate SendPacket — bodiless.
func TestItcStatusChargeByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ItcStatusCharge{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcStatusCharge body (v61): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcQueryCashRequest version=gms_v61 ida=0x5291cc
//
// ITC_QUERY_CASH_REQUEST (gms_v61 opcode 0xD6/214). CITC::TrySendQueryCashRequest
// @0x5291cc (inlined, renamed from sub_5291CC; nId 1001 query button);
// COutPacket(214) + immediate SendPacket, latch set after send — bodiless.
func TestItcQueryCashRequestByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := ItcQueryCashRequest{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcQueryCashRequest body (v61): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// ITC_OPERATION arms (gms_v61 dispatcher opcode 0xD7/215). Each arm's mode byte
// was verified from the v61 sender; bodies are byte-identical to gms_v83.

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterSale version=gms_v61 ida=0x528f35
//
// mode 2 register-fixed-price. CITC::OnRegisterSaleEntry @0x528f35 a2==0 branch:
// Encode1(2), item blob, Encode4 slotPos/quantity/price, Encode1 type/flag.
func TestItcOperationRegisterSaleByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
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
		t.Fatalf("RegisterSale (v61):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterAuction version=gms_v61 ida=0x528f35
//
// mode 0x12 register-auction. CITC::OnRegisterSaleEntry @0x528f35 a2==1 branch:
// Encode1(0x12), item blob, Encode4 slotPos/quantity/selector/buyNow, Encode1
// duration/flag, Encode4 increment.
func TestItcOperationRegisterAuctionByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
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
		t.Fatalf("RegisterAuction (v61):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSaleCurrentItem version=gms_v61 ida=0x52913b
//
// mode 3 want-ad offer. CITC::OnSaleCurrentItem @0x52913b: Encode1(3), Encode1
// type, Encode4 slotPos, item blob, Encode4 wishSerial.
func TestItcOperationSaleCurrentItemByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)
	got := pt.Encode(t, ctx, NewItcOperationSaleCurrentItem(0x03, 0x01, 7, asset, 2000000).Encode, nil)
	var want []byte
	want = append(want, 0x03, 0x01)
	want = append(want, le32(7)...)
	want = append(want, assetBytes...)
	want = append(want, le32(2000000)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("SaleCurrentItem (v61):\n got %v\nwant %v", got, want)
	}
}

// serialOnlyV61 asserts a mode+Encode4(nITCSN) arm for gms_v61.
func serialOnlyV61(t *testing.T, mode byte, got []byte) {
	t.Helper()
	var want []byte
	want = append(want, mode)
	want = append(want, le32(123456)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("serial-only mode 0x%02x (v61):\n got %v\nwant %v", mode, got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuy version=gms_v61 ida=0x529964
func TestItcOperationBuyByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	serialOnlyV61(t, 0x10, pt.Encode(t, ctx, NewItcOperationBuy(0x10, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyAuctionImm version=gms_v61 ida=0x5299d9
func TestItcOperationBuyAuctionImmByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	serialOnlyV61(t, 0x14, pt.Encode(t, ctx, NewItcOperationBuyAuctionImm(0x14, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelSale version=gms_v61 ida=0x529e54
func TestItcOperationCancelSaleByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	serialOnlyV61(t, 0x07, pt.Encode(t, ctx, NewItcOperationCancelSale(0x07, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationMoveLtoS version=gms_v61 ida=0x529efc
func TestItcOperationMoveLtoSByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	serialOnlyV61(t, 0x08, pt.Encode(t, ctx, NewItcOperationMoveLtoS(0x08, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSetZzim version=gms_v61 ida=0x529b6e
func TestItcOperationSetZzimByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	serialOnlyV61(t, 0x09, pt.Encode(t, ctx, NewItcOperationSetZzim(0x09, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyZzim version=gms_v61 ida=0x529be3
func TestItcOperationBuyZzimByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	serialOnlyV61(t, 0x11, pt.Encode(t, ctx, NewItcOperationBuyZzim(0x11, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationDeleteZzim version=gms_v61 ida=0x529c80
func TestItcOperationDeleteZzimByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	serialOnlyV61(t, 0x0A, pt.Encode(t, ctx, NewItcOperationDeleteZzim(0x0A, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationViewWish version=gms_v61 ida=0x529cf5
func TestItcOperationViewWishByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	serialOnlyV61(t, 0x0B, pt.Encode(t, ctx, NewItcOperationViewWish(0x0B, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyWish version=gms_v61 ida=0x529d6a
func TestItcOperationBuyWishByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	serialOnlyV61(t, 0x0C, pt.Encode(t, ctx, NewItcOperationBuyWish(0x0C, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelWish version=gms_v61 ida=0x529ddf
func TestItcOperationCancelWishByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	serialOnlyV61(t, 0x0D, pt.Encode(t, ctx, NewItcOperationCancelWish(0x0D, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationPlaceBid version=gms_v61 ida=0x549672
//
// mode 0x13 place-bid. CITCBidAuctionDlg::OnButtonClicked @0x549672 (nId==1
// confirm branch; send @0x549809): Encode1(0x13), Encode4 nITCSN/bidPrice/bidRange.
func TestItcOperationPlaceBidByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := pt.Encode(t, ctx, NewItcOperationPlaceBid(0x13, 123456, 5000, 100).Encode, nil)
	var want []byte
	want = append(want, 0x13)
	want = append(want, le32(123456)...)
	want = append(want, le32(5000)...)
	want = append(want, le32(100)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("PlaceBid (v61):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterWishEntry version=gms_v61 ida=0x52980f
//
// mode 4 register-wish-entry. CITC::OnRegisterWishEntry @0x52980f: Encode1(4),
// Encode4 itemId/price/count, Encode1 duration/feeOption, EncodeStr desc.
func TestItcOperationRegisterWishEntryByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
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
		t.Fatalf("RegisterWishEntry (v61):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategory version=gms_v61 ida=0x529560
func TestItcOperationChangedCategoryByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
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
		t.Fatalf("ChangedCategory (v61):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategorySub version=gms_v61 ida=0x529640
func TestItcOperationChangedCategorySubByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
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
		t.Fatalf("ChangedCategorySub (v61):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedPage version=gms_v61 ida=0x529730
func TestItcOperationChangedPageByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
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
		t.Fatalf("ChangedPage (v61):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationTabSearch version=gms_v61 ida=0x53e6a9
//
// mode 6 tab/search-by-name. CITCWnd_Tab::OnButtonClicked @0x53e6a9 (nId 1004
// char-name branch; send @0x53e741): Encode1(6), Encode4 category/categorySub/
// page/searchOption, EncodeStr name. 6-field body, NO sort bytes.
func TestItcOperationTabSearchByteOutput_v61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
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
		t.Fatalf("TabSearch (v61):\n got %v\nwant %v", got, want)
	}
}
