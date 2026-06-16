package clientbound

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

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
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			var out MtsItem
			pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
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
	ctx := pt.CreateContext("GMS", 95, 0)
	items := []MtsItem{mtsTestItem()}
	input := NewMtsResultGetItcListDone(0xAABBCCDD, 0x01020304, 0x05060708, 0x090A0B0C, 0x1F, 0x2E, items, 0x01)
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
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
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
	ctx := pt.CreateContext("GMS", 95, 0)
	items := []MtsItem{mtsTestItem()}
	input := NewMtsResultGetSearchItcListDone(0xAABBCCDD, 0x01020304, 0x05060708, 0x090A0B0C, items)
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
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
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
	ctx := pt.CreateContext("GMS", 95, 0)
	items := []MtsItem{mtsTestItem()}
	input := NewMtsResultGetUserPurchaseItemDone(items, 0xDEADBEEF, 0x01)
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
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
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
	ctx := pt.CreateContext("GMS", 95, 0)
	items := []MtsItem{mtsTestItem(), mtsTestItem()}
	input := NewMtsResultGetUserSaleItemDone(items)
	b := input.Encode(l, ctx)(nil)

	if b[0] != 0x23 {
		t.Fatalf("mode: got %#x want 0x23", b[0])
	}
	if got := le32(b[1:]); got != 2 {
		t.Errorf("totalCount: got %d want 2", got)
	}

	output := MtsResultGetUserSaleItemDone{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
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
	ctx := pt.CreateContext("GMS", 95, 0)
	items := []MtsItem{mtsTestItem()}
	input := NewMtsResultLoadWishSaleListDone(items)
	b := input.Encode(l, ctx)(nil)

	if b[0] != 0x2D {
		t.Fatalf("mode: got %#x want 0x2D", b[0])
	}
	if got := le32(b[1:]); got != 1 {
		t.Errorf("totalCount: got %d want 1", got)
	}

	output := MtsResultLoadWishSaleListDone{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.Mode() != 0x2D || len(output.Items()) != 1 {
		t.Errorf("round-trip mismatch: %+v", output)
	}
}

func le32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
