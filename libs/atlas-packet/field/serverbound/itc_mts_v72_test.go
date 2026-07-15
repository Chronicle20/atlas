package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// gms_v72 MTS/ITC (CITC) serverbound byte fixtures (task-113 v72 MTS port,
// GMS_v72.1_U_DEVM.exe, IDA port 13339). The codecs are version-agnostic; the
// v72 wire bodies are byte-identical to gms_v83 and every mode byte was
// re-verified from the v72 CITC send-sites (mostly inlined and renamed in the
// v72 IDB). Only the container opcodes differ: ENTER_MTS 0x9A/154,
// ITC_STATUS_CHARGE 0xEF/239, ITC_QUERY_CASH_REQUEST 0xF0/240, ITC_OPERATION
// 0xF1/241. itcTestAsset/le32 are defined in itc_operation_test.go (same
// package).

// packet-audit:verify packet=field/serverbound/FieldEnterMts version=gms_v72 ida=0x90c9bd
//
// ENTER_MTS (gms_v72 opcode 0x9A/154). CWvsContext::SendMigrateToITCRequest
// @0x90c9bd; send site COutPacket(154) @0x90cad6 then SendPacket with ZERO
// Encode calls — bodiless. Guest/map-flag guards emit chat/notice and return
// early; none writes the wire.
func TestEnterMtsByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := EnterMts{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("EnterMts body (v72): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcStatusCharge version=gms_v72 ida=0x561585
//
// ITC_STATUS_CHARGE (gms_v72 opcode 0xEF/239). CITC::OnStatusCharge @0x561585
// (inlined, renamed from sub_561585); latch set before COutPacket(239) +
// immediate SendPacket — bodiless.
func TestItcStatusChargeByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := ItcStatusCharge{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcStatusCharge body (v72): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcQueryCashRequest version=gms_v72 ida=0x561879
//
// ITC_QUERY_CASH_REQUEST (gms_v72 opcode 0xF0/240). CITC::TrySendQueryCashRequest
// @0x561879 (inlined, renamed from sub_561879); COutPacket(240) + immediate
// SendPacket, latch set after send — bodiless.
func TestItcQueryCashRequestByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := ItcQueryCashRequest{}.Encode(nil, ctx)(nil)
	if len(got) != 0 {
		t.Fatalf("ItcQueryCashRequest body (v72): got %d bytes %v, want 0 bytes (bodiless)", len(got), got)
	}
}

// ITC_OPERATION arms (gms_v72 dispatcher opcode 0xF1/241). Each arm's mode byte
// was verified from the v72 sender; bodies are byte-identical to gms_v83.

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterSale version=gms_v72 ida=0x5615e1
//
// mode 2 register-fixed-price. CITC::OnRegisterSaleEntry @0x5615e1 arg0==0 branch
// (send @0x56173d): Encode1(2), item blob, Encode4 slotPos/quantity/price,
// Encode1 type/flag.
func TestItcOperationRegisterSaleByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
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
		t.Fatalf("RegisterSale (v72):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterAuction version=gms_v72 ida=0x5615e1
//
// mode 0x12 register-auction. CITC::OnRegisterSaleEntry @0x5615e1 arg0==1 branch
// (send @0x561673): Encode1(0x12), item blob, Encode4 slotPos/quantity/selector/
// buyNow, Encode1 duration/flag, Encode4 increment.
func TestItcOperationRegisterAuctionByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
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
		t.Fatalf("RegisterAuction (v72):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSaleCurrentItem version=gms_v72 ida=0x5617ea
//
// mode 3 want-ad offer. CITC::OnSaleCurrentItem @0x5617ea: Encode1(3), Encode1
// type, Encode4 slotPos, item blob, Encode4 wishSerial.
func TestItcOperationSaleCurrentItemByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	asset := itcTestAsset()
	assetBytes := pt.Encode(t, ctx, asset.Encode, nil)
	got := pt.Encode(t, ctx, NewItcOperationSaleCurrentItem(0x03, 0x01, 7, asset, 2000000).Encode, nil)
	var want []byte
	want = append(want, 0x03, 0x01)
	want = append(want, le32(7)...)
	want = append(want, assetBytes...)
	want = append(want, le32(2000000)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("SaleCurrentItem (v72):\n got %v\nwant %v", got, want)
	}
}

// serialOnlyV72 asserts a mode+Encode4(nITCSN) arm for gms_v72.
func serialOnlyV72(t *testing.T, mode byte, got []byte) {
	t.Helper()
	var want []byte
	want = append(want, mode)
	want = append(want, le32(123456)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("serial-only mode 0x%02x (v72):\n got %v\nwant %v", mode, got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuy version=gms_v72 ida=0x562009
func TestItcOperationBuyByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	serialOnlyV72(t, 0x10, pt.Encode(t, ctx, NewItcOperationBuy(0x10, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyAuctionImm version=gms_v72 ida=0x56207c
func TestItcOperationBuyAuctionImmByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	serialOnlyV72(t, 0x14, pt.Encode(t, ctx, NewItcOperationBuyAuctionImm(0x14, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelSale version=gms_v72 ida=0x5624ec
func TestItcOperationCancelSaleByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	serialOnlyV72(t, 0x07, pt.Encode(t, ctx, NewItcOperationCancelSale(0x07, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationMoveLtoS version=gms_v72 ida=0x562595
func TestItcOperationMoveLtoSByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	serialOnlyV72(t, 0x08, pt.Encode(t, ctx, NewItcOperationMoveLtoS(0x08, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationSetZzim version=gms_v72 ida=0x56220f
func TestItcOperationSetZzimByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	serialOnlyV72(t, 0x09, pt.Encode(t, ctx, NewItcOperationSetZzim(0x09, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyZzim version=gms_v72 ida=0x562282
func TestItcOperationBuyZzimByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	serialOnlyV72(t, 0x11, pt.Encode(t, ctx, NewItcOperationBuyZzim(0x11, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationDeleteZzim version=gms_v72 ida=0x562320
func TestItcOperationDeleteZzimByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	serialOnlyV72(t, 0x0A, pt.Encode(t, ctx, NewItcOperationDeleteZzim(0x0A, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationViewWish version=gms_v72 ida=0x562393
func TestItcOperationViewWishByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	serialOnlyV72(t, 0x0B, pt.Encode(t, ctx, NewItcOperationViewWish(0x0B, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationBuyWish version=gms_v72 ida=0x562406
func TestItcOperationBuyWishByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	serialOnlyV72(t, 0x0C, pt.Encode(t, ctx, NewItcOperationBuyWish(0x0C, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationCancelWish version=gms_v72 ida=0x562479
func TestItcOperationCancelWishByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	serialOnlyV72(t, 0x0D, pt.Encode(t, ctx, NewItcOperationCancelWish(0x0D, 123456).Encode, nil))
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationPlaceBid version=gms_v72 ida=0x584a31
//
// mode 0x13 place-bid. CITCBidAuctionDlg::OnButtonClicked @0x584a31 (nId==1
// confirm branch): Encode1(0x13), Encode4 nITCSN/bidPrice/bidRange.
func TestItcOperationPlaceBidByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := pt.Encode(t, ctx, NewItcOperationPlaceBid(0x13, 123456, 5000, 100).Encode, nil)
	var want []byte
	want = append(want, 0x13)
	want = append(want, le32(123456)...)
	want = append(want, le32(5000)...)
	want = append(want, le32(100)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("PlaceBid (v72):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationRegisterWishEntry version=gms_v72 ida=0x561eb4
//
// mode 4 register-wish-entry. CITC::OnRegisterWishEntry @0x561eb4: Encode1(4),
// Encode4 itemId/price/count, Encode1 duration/feeOption, EncodeStr desc.
func TestItcOperationRegisterWishEntryByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
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
		t.Fatalf("RegisterWishEntry (v72):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategory version=gms_v72 ida=0x561c09
func TestItcOperationChangedCategoryByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
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
		t.Fatalf("ChangedCategory (v72):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedCategorySub version=gms_v72 ida=0x561ce8
func TestItcOperationChangedCategorySubByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
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
		t.Fatalf("ChangedCategorySub (v72):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationChangedPage version=gms_v72 ida=0x561dd7
func TestItcOperationChangedPageByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
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
		t.Fatalf("ChangedPage (v72):\n got %v\nwant %v", got, want)
	}
}

// packet-audit:verify packet=field/serverbound/FieldItcOperationTabSearch version=gms_v72 ida=0x578c91
//
// mode 6 tab/search-by-name. CITCWnd_Tab::OnButtonClicked @0x578c91 (btn 1004,
// char-name branch): Encode1(6), Encode4 category/categorySub/page/searchOption,
// EncodeStr name. 6-field body, NO sort bytes.
func TestItcOperationTabSearchByteOutput_v72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
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
		t.Fatalf("TabSearch (v72):\n got %v\nwant %v", got, want)
	}
}
