package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Per-mode body arms of CITC::OnNormalItemResult (MTS_OPERATION), task-096
// iteration 1. The body shapes are version-stable across gms_v83/v84/v87/v95
// (IDA-confirmed identical read order; only the dispatcher mode bytes and
// sub-handler addresses shift). jms_v185 has NO CITC op (VERSION-ABSENT) so it
// is not marked. The per-version ida= addresses below pin each arm's
// sub-handler in the matching dispatcher.

// MtsResultRegisterSaleEntryFailed (0x1E) — Decode1(reason); ONLY when
// reason==0x48 a trailing Decode2 sale-limit short. Decompiled identical in all
// four versions (v83 sub_5A4581 / v84 sub_5B4A38 / v87 0x5d4640 / v95 0x576b80).
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryFailed version=gms_v83 ida=0x5a4581
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryFailed version=gms_v84 ida=0x5b4a38
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryFailed version=gms_v87 ida=0x5d4640
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryFailed version=gms_v95 ida=0x576b80
func TestMtsResultRegisterSaleEntryFailedGolden(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 0)

	// reason != 0x48: only mode + reason on the wire (no trailing short).
	t.Run("PlainReason", func(t *testing.T) {
		input := NewMtsResultRegisterSaleEntryFailed(0x1E, 0x42, 0) // 'B' -> NoticeFailReason only
		expected := []byte{0x1E, 0x42}
		actual := test.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("golden mismatch: got %v want %v", actual, expected)
		}
	})

	// reason == 0x48: mode + reason + Decode2 sale-limit short (little-endian).
	t.Run("SaleLimitReason", func(t *testing.T) {
		input := NewMtsResultRegisterSaleEntryFailed(0x1E, 0x48, 0x0064) // 'H', limit 100
		expected := []byte{0x1E, 0x48, 0x64, 0x00}
		actual := test.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("golden mismatch: got %v want %v", actual, expected)
		}
	})
}

func TestMtsResultRegisterSaleEntryFailedRoundTrip(t *testing.T) {
	for _, in := range []MtsResultRegisterSaleEntryFailed{
		NewMtsResultRegisterSaleEntryFailed(0x1E, 0x42, 0),
		NewMtsResultRegisterSaleEntryFailed(0x1E, 0x48, 0x1234),
	} {
		input := in
		for _, v := range test.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				output := MtsResultRegisterSaleEntryFailed{}
				test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Mode() != input.Mode() {
					t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
				}
				if output.Reason() != input.Reason() {
					t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
				}
				if output.SaleLimit() != input.SaleLimit() {
					t.Errorf("saleLimit: got %v, want %v", output.SaleLimit(), input.SaleLimit())
				}
			})
		}
	}
}

// MtsResultSuccessBidInfo (0x3E) — Decode1(soldFlag) + Decode4(itemId); ONLY when
// itemId>0 a trailing Decode4(price) + DecodeBuffer(8) FILETIME contract date.
// Decompiled identical in all four versions (v83 sub_5A52DE / v84 sub_5B5795 /
// v87 0x5d5398 / v95 0x577000).
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultSuccessBidInfo version=gms_v83 ida=0x5a52de
// packet-audit:verify packet=field/clientbound/FieldMtsResultSuccessBidInfo version=gms_v84 ida=0x5b5795
// packet-audit:verify packet=field/clientbound/FieldMtsResultSuccessBidInfo version=gms_v87 ida=0x5d5398
// packet-audit:verify packet=field/clientbound/FieldMtsResultSuccessBidInfo version=gms_v95 ida=0x577000
func TestMtsResultSuccessBidInfoGolden(t *testing.T) {
	ctx := test.CreateContext("GMS", 95, 0)

	// itemId <= 0: body ends after mode + soldFlag + itemId (no notice path).
	t.Run("NoItem", func(t *testing.T) {
		input := NewMtsResultSuccessBidInfo(0x3E, 1, 0, 0, [8]byte{})
		expected := []byte{0x3E, 0x01, 0x00, 0x00, 0x00, 0x00}
		actual := test.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("golden mismatch: got %v want %v", actual, expected)
		}
	})

	// itemId > 0: mode + soldFlag + itemId + price + 8-byte FILETIME buffer.
	t.Run("WithItem", func(t *testing.T) {
		date := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
		input := NewMtsResultSuccessBidInfo(0x3E, 1, 0x00204FCE, 0x000F4240, date) // itemId 2117070, price 1,000,000
		expected := []byte{
			0x3E,                   // mode
			0x01,                   // soldFlag
			0xCE, 0x4F, 0x20, 0x00, // itemId 0x00204FCE little-endian
			0x40, 0x42, 0x0F, 0x00, // price 0x000F4240 little-endian
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, // contract date buffer
		}
		actual := test.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("golden mismatch: got %v want %v", actual, expected)
		}
	})
}

func TestMtsResultSuccessBidInfoRoundTrip(t *testing.T) {
	date := [8]byte{0xDE, 0xAD, 0xBE, 0xEF, 0x11, 0x22, 0x33, 0x44}
	for _, in := range []MtsResultSuccessBidInfo{
		NewMtsResultSuccessBidInfo(0x3E, 0, 0, 0, [8]byte{}),
		NewMtsResultSuccessBidInfo(0x3E, 1, 0x00204FCE, 0x000F4240, date),
	} {
		input := in
		for _, v := range test.Variants {
			t.Run(v.Name, func(t *testing.T) {
				ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
				output := MtsResultSuccessBidInfo{}
				test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
				if output.Mode() != input.Mode() {
					t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
				}
				if output.SoldFlag() != input.SoldFlag() {
					t.Errorf("soldFlag: got %v, want %v", output.SoldFlag(), input.SoldFlag())
				}
				if output.ItemId() != input.ItemId() {
					t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
				}
				if output.Price() != input.Price() {
					t.Errorf("price: got %v, want %v", output.Price(), input.Price())
				}
				if output.ContractDate() != input.ContractDate() {
					t.Errorf("contractDate: got %v, want %v", output.ContractDate(), input.ContractDate())
				}
			})
		}
	}
}

// List/item-blob body arms of CITC::OnNormalItemResult (MTS_OPERATION),
// task-096 iteration 6 (FINAL). Each arm embeds one or more ITCITEM entries; an
// ITCITEM wraps a GW_ItemSlotBase item blob (model.Asset codec) plus an MTS
// trailer (meso/contract/bid metadata). The read order is decompile-confirmed
// version-stable across gms_v83/v84/v87/v95 (loop count, item-blob, and any
// leading/trailing scalars identical; only the dispatcher mode bytes and
// sub-handler addresses shift). jms_v185 has NO CITC op (VERSION-ABSENT) so it
// is not marked.

// mtsTestItem builds one ITCITEM fixture: a stackable USE item blob (so the
// GW_ItemSlotBase codec round-trips deterministically) plus the MTS trailer.
func mtsTestItem() MtsItem {
	asset := model.NewAsset(true, 0, 2000000, time.Time{}).SetStackableInfo(5, 0, 0)
	var date [8]byte
	copy(date[:], []byte{1, 2, 3, 4, 5, 6, 7, 8})
	return NewMtsItem(
		asset,
		0x11111111, // nITCSN
		0x22222222, // nPrice
		0x33333333, // nContractFee
		"txid",     // sContractFeeTxId
		"rbid",     // sRollbackUsageID
		date,       // ftITCDateExpired
		"user",     // sUserID
		"game",     // sGameID
		"hello",    // sComment
		0x44444444, // nBidCount
		0x55555555, // nBidRange
		0x66666666, // nBidPrice
		0x77777777, // nMinPrice
		0x10101010, // nMaxPrice
		0x20202020, // nUnitPrice
		0x3030,     // nProcessStatus
	)
}

// TestMtsItemRoundTrip proves the ITCITEM trailer + embedded GW_ItemSlotBase
// blob round-trip byte-exactly across every variant.
func TestMtsItemRoundTrip(t *testing.T) {
	in := mtsTestItem()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			var out MtsItem
			test.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
			if out.ItcSn() != in.ItcSn() {
				t.Errorf("itcSn: got %x want %x", out.ItcSn(), in.ItcSn())
			}
			if out.Price() != in.Price() {
				t.Errorf("price: got %x want %x", out.Price(), in.Price())
			}
			if out.ContractFeeTx() != in.ContractFeeTx() {
				t.Errorf("contractFeeTx: got %q want %q", out.ContractFeeTx(), in.ContractFeeTx())
			}
			if out.Comment() != in.Comment() {
				t.Errorf("comment: got %q want %q", out.Comment(), in.Comment())
			}
			if out.ProcessStatus() != in.ProcessStatus() {
				t.Errorf("processStatus: got %x want %x", out.ProcessStatus(), in.ProcessStatus())
			}
		})
	}
}

// MtsResultGetItcListDone (0x15) — Decode4 catItemCnt, Decode4 pageItemCnt,
// Decode4 category, Decode4 subCategory, Decode4 page, Decode1 sortType,
// Decode1 sortColumn, pageItemCnt × ITCITEM, Decode1 requestSent. The leading
// header bytes are byte-asserted; the item blob + round-trip prove the rest.
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetItcListDone version=gms_v83 ida=0x5a48af
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetItcListDone version=gms_v84 ida=0x5b4d9f
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetItcListDone version=gms_v87 ida=0x5d499f
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetItcListDone version=gms_v95 ida=0x576500
func TestMtsResultGetItcListDoneGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 0)
	items := []MtsItem{mtsTestItem()}
	input := NewMtsResultGetItcListDone(0x15, 0xAABBCCDD, 0x01020304, 0x05060708, 0x090A0B0C, 0x1F, 0x2E, items, 0x01)
	b := input.Encode(l, ctx)(nil)

	// offset 0: mode 0x15
	if b[0] != 0x15 {
		t.Fatalf("mode: got %#x want 0x15", b[0])
	}
	// offset 1..4: categoryItemCnt (LE 0xAABBCCDD)
	if got := le32(b[1:]); got != 0xAABBCCDD {
		t.Errorf("categoryItemCnt: got %#x want 0xAABBCCDD", got)
	}
	// offset 5..8: pageItemCnt == len(items) == 1
	if got := le32(b[5:]); got != 1 {
		t.Errorf("pageItemCnt: got %d want 1", got)
	}
	// offset 9..12: category
	if got := le32(b[9:]); got != 0x01020304 {
		t.Errorf("category: got %#x want 0x01020304", got)
	}
	// offset 13..16: subCategory
	if got := le32(b[13:]); got != 0x05060708 {
		t.Errorf("subCategory: got %#x want 0x05060708", got)
	}
	// offset 17..20: page
	if got := le32(b[17:]); got != 0x090A0B0C {
		t.Errorf("page: got %#x want 0x090A0B0C", got)
	}
	// offset 21: sortType, offset 22: sortColumn
	if b[21] != 0x1F || b[22] != 0x2E {
		t.Errorf("sort bytes: got %#x %#x want 0x1F 0x2E", b[21], b[22])
	}
	// last byte: requestSent
	if b[len(b)-1] != 0x01 {
		t.Errorf("requestSent: got %#x want 0x01", b[len(b)-1])
	}

	output := MtsResultGetItcListDone{}
	test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Mode() != 0x15 || output.CategoryItemCnt() != 0xAABBCCDD ||
		output.Category() != 0x01020304 || output.SubCategory() != 0x05060708 ||
		output.Page() != 0x090A0B0C || output.SortType() != 0x1F ||
		output.SortColumn() != 0x2E || output.RequestSent() != 0x01 ||
		len(output.Items()) != 1 {
		t.Errorf("round-trip mismatch: %+v", output)
	}
}

// MtsResultGetSearchItcListDone (0x17) — Decode4 catItemCnt, Decode4
// pageItemCnt, Decode4 category, Decode4 subCategory, Decode4 page, pageItemCnt
// × ITCITEM. NO sort bytes, NO trailing requestSent byte.
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetSearchItcListDone version=gms_v83 ida=0x5a4a02
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetSearchItcListDone version=gms_v84 ida=0x5b4ef2
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetSearchItcListDone version=gms_v87 ida=0x5d4af2
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetSearchItcListDone version=gms_v95 ida=0x5766e0
func TestMtsResultGetSearchItcListDoneGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 0)
	items := []MtsItem{mtsTestItem()}
	input := NewMtsResultGetSearchItcListDone(0x17, 0xAABBCCDD, 0x01020304, 0x05060708, 0x090A0B0C, items)
	b := input.Encode(l, ctx)(nil)

	if b[0] != 0x17 {
		t.Fatalf("mode: got %#x want 0x17", b[0])
	}
	if got := le32(b[1:]); got != 0xAABBCCDD {
		t.Errorf("categoryItemCnt: got %#x", got)
	}
	if got := le32(b[5:]); got != 1 {
		t.Errorf("pageItemCnt: got %d want 1", got)
	}
	if got := le32(b[9:]); got != 0x01020304 {
		t.Errorf("category: got %#x", got)
	}
	if got := le32(b[13:]); got != 0x05060708 {
		t.Errorf("subCategory: got %#x", got)
	}
	if got := le32(b[17:]); got != 0x090A0B0C {
		t.Errorf("page: got %#x", got)
	}
	// offset 21 onward is the ITCITEM blob — NO sort bytes here (unlike 0x15).
	// The first byte of the item is the GW_ItemSlotBase type discriminator.

	output := MtsResultGetSearchItcListDone{}
	test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Mode() != 0x17 || output.CategoryItemCnt() != 0xAABBCCDD ||
		output.Category() != 0x01020304 || output.SubCategory() != 0x05060708 ||
		output.Page() != 0x090A0B0C || len(output.Items()) != 1 {
		t.Errorf("round-trip mismatch: %+v", output)
	}
}

// MtsResultGetUserPurchaseItemDone (0x21) — Decode4 totalCount, totalCount ×
// ITCITEM, Decode4 limitedCount, Decode1 requestSent.
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserPurchaseItemDone version=gms_v83 ida=0x5a4af3
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserPurchaseItemDone version=gms_v84 ida=0x5b4fe3
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserPurchaseItemDone version=gms_v87 ida=0x5d4be3
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserPurchaseItemDone version=gms_v95 ida=0x576cf0
func TestMtsResultGetUserPurchaseItemDoneGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 0)
	items := []MtsItem{mtsTestItem()}
	input := NewMtsResultGetUserPurchaseItemDone(0x21, items, 0xDEADBEEF, 0x01)
	b := input.Encode(l, ctx)(nil)

	if b[0] != 0x21 {
		t.Fatalf("mode: got %#x want 0x21", b[0])
	}
	if got := le32(b[1:]); got != 1 {
		t.Errorf("totalCount: got %d want 1", got)
	}
	// limitedCount (Decode4) + requestSent (Decode1) are the trailing 5 bytes.
	if got := le32(b[len(b)-5:]); got != 0xDEADBEEF {
		t.Errorf("limitedCount: got %#x want 0xDEADBEEF", got)
	}
	if b[len(b)-1] != 0x01 {
		t.Errorf("requestSent: got %#x want 0x01", b[len(b)-1])
	}

	output := MtsResultGetUserPurchaseItemDone{}
	test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Mode() != 0x21 || output.LimitedCount() != 0xDEADBEEF ||
		output.RequestSent() != 0x01 || len(output.Items()) != 1 {
		t.Errorf("round-trip mismatch: %+v", output)
	}
}

// MtsResultGetUserSaleItemDone (0x23) — Decode4 totalCount, totalCount ×
// ITCITEM. No trailing fields.
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserSaleItemDone version=gms_v83 ida=0x5a4c57
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserSaleItemDone version=gms_v84 ida=0x5b5147
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserSaleItemDone version=gms_v87 ida=0x5d4d47
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserSaleItemDone version=gms_v95 ida=0x576870
func TestMtsResultGetUserSaleItemDoneGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 0)
	items := []MtsItem{mtsTestItem(), mtsTestItem()}
	input := NewMtsResultGetUserSaleItemDone(0x23, items)
	b := input.Encode(l, ctx)(nil)

	if b[0] != 0x23 {
		t.Fatalf("mode: got %#x want 0x23", b[0])
	}
	if got := le32(b[1:]); got != 2 {
		t.Errorf("totalCount: got %d want 2", got)
	}

	output := MtsResultGetUserSaleItemDone{}
	test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Mode() != 0x23 || len(output.Items()) != 2 {
		t.Errorf("round-trip mismatch: %+v", output)
	}
}

// MtsResultLoadWishSaleListDone (0x2D) — Decode4 totalCount, totalCount ×
// ITCITEM. No trailing fields.
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListDone version=gms_v83 ida=0x5a4ec6
// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListDone version=gms_v84 ida=0x5b53b6
// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListDone version=gms_v87 ida=0x5d4fb9
// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListDone version=gms_v95 ida=0x5769a0
func TestMtsResultLoadWishSaleListDoneGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 0)
	items := []MtsItem{mtsTestItem()}
	input := NewMtsResultLoadWishSaleListDone(0x2D, items)
	b := input.Encode(l, ctx)(nil)

	if b[0] != 0x2D {
		t.Fatalf("mode: got %#x want 0x2D", b[0])
	}
	if got := le32(b[1:]); got != 1 {
		t.Errorf("totalCount: got %d want 1", got)
	}

	output := MtsResultLoadWishSaleListDone{}
	test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Mode() != 0x2D || len(output.Items()) != 1 {
		t.Errorf("round-trip mismatch: %+v", output)
	}
}

func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

// Discrete per-mode golden + round-trip tests for the notice-only ("Empty-shape")
// arms of CITC::OnNormalItemResult (MTS_OPERATION). Each arm fixes its own mode
// byte and writes exactly that byte (the sub-handler reads NOTHING after the
// dispatcher Decode1(mode) — StringPool::GetString + CUtilDlg::Notice only). The
// mode bytes are version-stable; per-version sub-handler addresses are cited in
// each verify marker (dispatcher: v83 0x5a4311 / v84 0x5b47c8 / v87 0x5d43d0 /
// v95 0x5771d0). jms_v185 has NO CITC op (VERSION-ABSENT, unmarked).

// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryDone version=gms_v83 ida=0x5a4674
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryDone version=gms_v84 ida=0x5b4b64
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryDone version=gms_v87 ida=0x5d4748
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterSaleEntryDone version=gms_v95 ida=0x575cd0
func TestMtsResultRegisterSaleEntryDone(t *testing.T) {
	input := NewMtsResultRegisterSaleEntryDone(0x1D)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x1D} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultRegisterSaleEntryDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishDone version=gms_v83 ida=0x5a46b2
// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishDone version=gms_v84 ida=0x5b4ba2
// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishDone version=gms_v87 ida=0x5d4786
// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishDone version=gms_v95 ida=0x575d20
func TestMtsResultSaleCurrentItemToWishDone(t *testing.T) {
	input := NewMtsResultSaleCurrentItemToWishDone(0x1F)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x1F} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultSaleCurrentItemToWishDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemDone version=gms_v83 ida=0x5a4d14
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemDone version=gms_v84 ida=0x5b5204
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemDone version=gms_v87 ida=0x5d4e04
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemDone version=gms_v95 ida=0x576030
func TestMtsResultCancelSaleItemDone(t *testing.T) {
	input := NewMtsResultCancelSaleItemDone(0x25)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x25} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultCancelSaleItemDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimDone version=gms_v83 ida=0x5a4dfc
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimDone version=gms_v84 ida=0x5b52ec
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimDone version=gms_v87 ida=0x5d4eef
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimDone version=gms_v95 ida=0x576140
func TestMtsResultSetZzimDone(t *testing.T) {
	input := NewMtsResultSetZzimDone(0x29)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x29} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultSetZzimDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimFailed version=gms_v83 ida=0x5a4e31
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimFailed version=gms_v84 ida=0x5b5321
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimFailed version=gms_v87 ida=0x5d4f24
// packet-audit:verify packet=field/clientbound/FieldMtsResultSetZzimFailed version=gms_v95 ida=0x576180
func TestMtsResultSetZzimFailed(t *testing.T) {
	input := NewMtsResultSetZzimFailed(0x2A)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x2A} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultSetZzimFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimDone version=gms_v83 ida=0x5a4e66
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimDone version=gms_v84 ida=0x5b5356
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimDone version=gms_v87 ida=0x5d4f59
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimDone version=gms_v95 ida=0x5761c0
func TestMtsResultDeleteZzimDone(t *testing.T) {
	input := NewMtsResultDeleteZzimDone(0x2B)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x2B} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultDeleteZzimDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimFailed version=gms_v83 ida=0x5a4e91
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimFailed version=gms_v84 ida=0x5b5381
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimFailed version=gms_v87 ida=0x5d4f84
// packet-audit:verify packet=field/clientbound/FieldMtsResultDeleteZzimFailed version=gms_v95 ida=0x5761f0
func TestMtsResultDeleteZzimFailed(t *testing.T) {
	input := NewMtsResultDeleteZzimFailed(0x2C)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x2C} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultDeleteZzimFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListFailed version=gms_v83 ida=0x5a4fdc
// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListFailed version=gms_v84 ida=0x5b54cc
// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListFailed version=gms_v87 ida=0x5d50cf
// packet-audit:verify packet=field/clientbound/FieldMtsResultLoadWishSaleListFailed version=gms_v95 ida=0x576230
func TestMtsResultLoadWishSaleListFailed(t *testing.T) {
	input := NewMtsResultLoadWishSaleListFailed(0x2E)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x2E} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultLoadWishSaleListFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishDone version=gms_v83 ida=0x5a5011
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishDone version=gms_v84 ida=0x5b5501
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishDone version=gms_v87 ida=0x5d5104
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishDone version=gms_v95 ida=0x576270
func TestMtsResultBuyWishDone(t *testing.T) {
	input := NewMtsResultBuyWishDone(0x2F)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x2F} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyWishDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishFailed version=gms_v83 ida=0x5a503c
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishFailed version=gms_v84 ida=0x5b552c
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishFailed version=gms_v87 ida=0x5d512f
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyWishFailed version=gms_v95 ida=0x5762a0
func TestMtsResultBuyWishFailed(t *testing.T) {
	input := NewMtsResultBuyWishFailed(0x30)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x30} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyWishFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishDone version=gms_v83 ida=0x5a5071
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishDone version=gms_v84 ida=0x5b5561
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishDone version=gms_v87 ida=0x5d5164
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishDone version=gms_v95 ida=0x5762e0
func TestMtsResultCancelWishDone(t *testing.T) {
	input := NewMtsResultCancelWishDone(0x31)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x31} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultCancelWishDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishFailed version=gms_v83 ida=0x5a50df
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishFailed version=gms_v84 ida=0x5b5596
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishFailed version=gms_v87 ida=0x5d5199
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelWishFailed version=gms_v95 ida=0x576320
func TestMtsResultCancelWishFailed(t *testing.T) {
	input := NewMtsResultCancelWishFailed(0x32)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x32} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultCancelWishFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemDone version=gms_v83 ida=0x5a5114
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemDone version=gms_v84 ida=0x5b55cb
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemDone version=gms_v87 ida=0x5d51ce
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemDone version=gms_v95 ida=0x576360
func TestMtsResultBuyItemDone(t *testing.T) {
	input := NewMtsResultBuyItemDone(0x33)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x33} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyItemDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemFailed version=gms_v83 ida=0x5a513f
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemFailed version=gms_v84 ida=0x5b55f6
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemFailed version=gms_v87 ida=0x5d51f9
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyItemFailed version=gms_v95 ida=0x576390
func TestMtsResultBuyItemFailed(t *testing.T) {
	input := NewMtsResultBuyItemFailed(0x34)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x34} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyItemFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemDone version=gms_v83 ida=0x5a5174
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemDone version=gms_v84 ida=0x5b562b
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemDone version=gms_v87 ida=0x5d522e
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemDone version=gms_v95 ida=0x5763d0
func TestMtsResultBuyZzimItemDone(t *testing.T) {
	input := NewMtsResultBuyZzimItemDone(0x35)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x35} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyZzimItemDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemFailed version=gms_v83 ida=0x5a519f
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemFailed version=gms_v84 ida=0x5b5656
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemFailed version=gms_v87 ida=0x5d5259
// packet-audit:verify packet=field/clientbound/FieldMtsResultBuyZzimItemFailed version=gms_v95 ida=0x576400
func TestMtsResultBuyZzimItemFailed(t *testing.T) {
	input := NewMtsResultBuyZzimItemFailed(0x36)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x36} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBuyZzimItemFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemDone version=gms_v83 ida=0x5a51d4
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemDone version=gms_v84 ida=0x5b568b
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemDone version=gms_v87 ida=0x5d528e
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemDone version=gms_v95 ida=0x576440
func TestMtsResultRegisterWishItemDone(t *testing.T) {
	input := NewMtsResultRegisterWishItemDone(0x37)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x37} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultRegisterWishItemDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemFailed version=gms_v83 ida=0x5a5209
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemFailed version=gms_v84 ida=0x5b56c0
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemFailed version=gms_v87 ida=0x5d52c3
// packet-audit:verify packet=field/clientbound/FieldMtsResultRegisterWishItemFailed version=gms_v95 ida=0x576480
func TestMtsResultRegisterWishItemFailed(t *testing.T) {
	input := NewMtsResultRegisterWishItemFailed(0x38)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x38} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultRegisterWishItemFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultBidAuctionFailed version=gms_v83 ida=0x5a5444
// packet-audit:verify packet=field/clientbound/FieldMtsResultBidAuctionFailed version=gms_v84 ida=0x5b58fb
// packet-audit:verify packet=field/clientbound/FieldMtsResultBidAuctionFailed version=gms_v87 ida=0x5d54fe
// packet-audit:verify packet=field/clientbound/FieldMtsResultBidAuctionFailed version=gms_v95 ida=0x5764c0
func TestMtsResultBidAuctionFailed(t *testing.T) {
	input := NewMtsResultBidAuctionFailed(0x3C)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x3C} // dispatcher mode byte; sub-handler reads no further fields
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultBidAuctionFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// Discrete per-mode golden + round-trip tests for the "Reason-shape" arms of
// CITC::OnNormalItemResult (MTS_OPERATION). Each arm fixes its own mode byte and
// writes the mode byte THEN one Decode1 fail-reason byte (the sub-handler routes
// the reason to NoticeFailReason / a reason-keyed StringPool notice and reads
// NOTHING further). The mode bytes are version-stable; per-version sub-handler
// addresses are cited in each verify marker (dispatcher: v83 0x5a4311 /
// v84 0x5b47c8 / v87 0x5d43d0 / v95 0x5771d0). jms_v185 has NO CITC op
// (VERSION-ABSENT, unmarked).

// packet-audit:verify packet=field/clientbound/FieldMtsResultGetItcListFailed version=gms_v83 ida=0x5a4882
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetItcListFailed version=gms_v84 ida=0x5b4d72
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetItcListFailed version=gms_v87 ida=0x5d4972
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetItcListFailed version=gms_v95 ida=0x575f70
func TestMtsResultGetItcListFailed(t *testing.T) {
	input := NewMtsResultGetItcListFailed(0x16, 0x49) // reason 73 = the transfer-field branch value
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x16, 0x49} // dispatcher mode byte (0x16) + Decode1 reason
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultGetItcListFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Reason() != input.Reason() {
				t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultGetSearchItcListFailed version=gms_v83 ida=0x5a49e3
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetSearchItcListFailed version=gms_v84 ida=0x5b4ed3
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetSearchItcListFailed version=gms_v87 ida=0x5d4ad3
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetSearchItcListFailed version=gms_v95 ida=0x575fa0
func TestMtsResultGetSearchItcListFailed(t *testing.T) {
	input := NewMtsResultGetSearchItcListFailed(0x18, 0x51)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x18, 0x51} // dispatcher mode byte (0x18) + Decode1 reason
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultGetSearchItcListFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Reason() != input.Reason() {
				t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishFailed version=gms_v83 ida=0x5a46f0
// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishFailed version=gms_v84 ida=0x5b4be0
// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishFailed version=gms_v87 ida=0x5d47c4
// packet-audit:verify packet=field/clientbound/FieldMtsResultSaleCurrentItemToWishFailed version=gms_v95 ida=0x575d70
func TestMtsResultSaleCurrentItemToWishFailed(t *testing.T) {
	input := NewMtsResultSaleCurrentItemToWishFailed(0x20, 0x50)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x20, 0x50} // dispatcher mode byte (0x20) + Decode1 reason
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultSaleCurrentItemToWishFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Reason() != input.Reason() {
				t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserPurchaseItemFailed version=gms_v83 ida=0x5a4c2a
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserPurchaseItemFailed version=gms_v84 ida=0x5b511a
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserPurchaseItemFailed version=gms_v87 ida=0x5d4d1a
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserPurchaseItemFailed version=gms_v95 ida=0x575fd0
func TestMtsResultGetUserPurchaseItemFailed(t *testing.T) {
	input := NewMtsResultGetUserPurchaseItemFailed(0x22, 0x49)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x22, 0x49} // dispatcher mode byte (0x22) + Decode1 reason
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultGetUserPurchaseItemFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Reason() != input.Reason() {
				t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserSaleItemFailed version=gms_v83 ida=0x5a4ce7
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserSaleItemFailed version=gms_v84 ida=0x5b51d7
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserSaleItemFailed version=gms_v87 ida=0x5d4dd7
// packet-audit:verify packet=field/clientbound/FieldMtsResultGetUserSaleItemFailed version=gms_v95 ida=0x576000
func TestMtsResultGetUserSaleItemFailed(t *testing.T) {
	input := NewMtsResultGetUserSaleItemFailed(0x24, 0x49)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x24, 0x49} // dispatcher mode byte (0x24) + Decode1 reason
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultGetUserSaleItemFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Reason() != input.Reason() {
				t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemFailed version=gms_v83 ida=0x5a4d49
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemFailed version=gms_v84 ida=0x5b5239
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemFailed version=gms_v87 ida=0x5d4e39
// packet-audit:verify packet=field/clientbound/FieldMtsResultCancelSaleItemFailed version=gms_v95 ida=0x576070
func TestMtsResultCancelSaleItemFailed(t *testing.T) {
	input := NewMtsResultCancelSaleItemFailed(0x26, 0x42)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x26, 0x42} // dispatcher mode byte (0x26) + Decode1 reason
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultCancelSaleItemFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Reason() != input.Reason() {
				t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultMoveItcPurchaseItemLtoSFailed version=gms_v83 ida=0x5a4dcf
// packet-audit:verify packet=field/clientbound/FieldMtsResultMoveItcPurchaseItemLtoSFailed version=gms_v84 ida=0x5b52bf
// packet-audit:verify packet=field/clientbound/FieldMtsResultMoveItcPurchaseItemLtoSFailed version=gms_v87 ida=0x5d4ec2
// packet-audit:verify packet=field/clientbound/FieldMtsResultMoveItcPurchaseItemLtoSFailed version=gms_v95 ida=0x576110
func TestMtsResultMoveItcPurchaseItemLtoSFailed(t *testing.T) {
	input := NewMtsResultMoveItcPurchaseItemLtoSFailed(0x28, 0x41) // reason 65 = the transfer-field re-send branch value
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{0x28, 0x41} // dispatcher mode byte (0x28) + Decode1 reason
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultMoveItcPurchaseItemLtoSFailed{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Reason() != input.Reason() {
				t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
			}
		})
	}
}

// Discrete per-mode golden + round-trip tests for the "TwoInts-shape" arms of
// CITC::OnNormalItemResult (MTS_OPERATION). Each arm fixes its own mode byte and
// writes the mode byte THEN two Decode4 ints (little-endian via WriteInt). The
// mode bytes are version-stable; per-version sub-handler addresses are cited in
// each verify marker (dispatcher: v83 0x5a4311 / v84 0x5b47c8 / v87 0x5d43d0 /
// v95 0x5771d0). jms_v185 has NO CITC op (VERSION-ABSENT, unmarked).

// packet-audit:verify packet=field/clientbound/FieldMtsResultMoveItcPurchaseItemLtoSDone version=gms_v83 ida=0x5a4d68
// packet-audit:verify packet=field/clientbound/FieldMtsResultMoveItcPurchaseItemLtoSDone version=gms_v84 ida=0x5b5258
// packet-audit:verify packet=field/clientbound/FieldMtsResultMoveItcPurchaseItemLtoSDone version=gms_v87 ida=0x5d4e58
// packet-audit:verify packet=field/clientbound/FieldMtsResultMoveItcPurchaseItemLtoSDone version=gms_v95 ida=0x5760a0
func TestMtsResultMoveItcPurchaseItemLtoSDone(t *testing.T) {
	// Sub-handler decompile (v95 0x5760a0): Decode4(tab) -> SetTab(tab-1),
	// Decode4(selectedNo). The codec writes them little-endian via WriteInt.
	input := NewMtsResultMoveItcPurchaseItemLtoSDone(0x27, 0x00000003, 0x0000000A)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{
		0x27,                   // dispatcher mode byte (0x27)
		0x03, 0x00, 0x00, 0x00, // Decode4 tab
		0x0A, 0x00, 0x00, 0x00, // Decode4 selectedNo
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultMoveItcPurchaseItemLtoSDone{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Tab() != input.Tab() {
				t.Errorf("tab: got %v, want %v", output.Tab(), input.Tab())
			}
			if output.SelectedNo() != input.SelectedNo() {
				t.Errorf("selectedNo: got %v, want %v", output.SelectedNo(), input.SelectedNo())
			}
		})
	}
}

// packet-audit:verify packet=field/clientbound/FieldMtsResultNotifyCancelWishResult version=gms_v83 ida=0x5a523e
// packet-audit:verify packet=field/clientbound/FieldMtsResultNotifyCancelWishResult version=gms_v84 ida=0x5b56f5
// packet-audit:verify packet=field/clientbound/FieldMtsResultNotifyCancelWishResult version=gms_v87 ida=0x5d52f8
// packet-audit:verify packet=field/clientbound/FieldMtsResultNotifyCancelWishResult version=gms_v95 ida=0x576f00
func TestMtsResultNotifyCancelWishResult(t *testing.T) {
	// Sub-handler decompile (v95 0x576f00): Decode4(countA) Decode4(countB);
	// each >0 guards a StringPool notice. Read order is Decode4 then Decode4.
	input := NewMtsResultNotifyCancelWishResult(0x3D, 0x00000005, 0x00000002)
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{
		0x3D,                   // dispatcher mode byte (0x3D)
		0x05, 0x00, 0x00, 0x00, // Decode4 countA
		0x02, 0x00, 0x00, 0x00, // Decode4 countB
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			vctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultNotifyCancelWishResult{}
			test.RoundTrip(t, vctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.CountA() != input.CountA() {
				t.Errorf("countA: got %v, want %v", output.CountA(), input.CountA())
			}
			if output.CountB() != input.CountB() {
				t.Errorf("countB: got %v, want %v", output.CountB(), input.CountB())
			}
		})
	}
}
