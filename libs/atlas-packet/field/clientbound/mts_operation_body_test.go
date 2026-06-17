package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Per-mode body arms of CITC::OnNormalItemResult (MTS_OPERATION), task-096
// iteration 1. The body shapes are version-stable across gms_v83/v84/v87/v95
// (IDA-confirmed identical read order; only the dispatcher mode bytes and
// sub-handler addresses shift). jms_v185 has NO CITC op (VERSION-ABSENT) so it
// is not marked. The per-version ida= addresses below pin each arm's
// sub-handler in the matching dispatcher.

// MtsResultReason arms — sub-handler reads a single Decode1 fail-reason byte
// after the dispatcher mode byte. GET_ITC_LIST_FAILED (0x16) is the pinned
// representative; SALE_CURRENT_ITEM_TO_WISH_FAILED (0x20) shares the identical
// mode + Decode1(reason) wire shape.
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultReason version=gms_v83 ida=0x5a4882
// packet-audit:verify packet=field/clientbound/FieldMtsResultReason version=gms_v84 ida=0x5b4d72
// packet-audit:verify packet=field/clientbound/FieldMtsResultReason version=gms_v87 ida=0x5d4972
// packet-audit:verify packet=field/clientbound/FieldMtsResultReason version=gms_v95 ida=0x575f70
func TestMtsResultReasonGolden(t *testing.T) {
	// mode 0x16 = GET_ITC_LIST_FAILED. Sub-handler decompile (v95 0x575f70):
	// Decode1(reason) -> NoticeFailReason(reason). The wire after the dispatcher
	// mode byte is exactly one Decode1 reason byte.
	ctx := test.CreateContext("GMS", 95, 0)
	cases := []struct {
		name   string
		mode   byte
		reason byte
	}{
		{"GetITCListFailed", 0x16, 0x49}, // reason 73 = the transfer-field branch value
		{"SaleCurrentItemToWishFailed", 0x20, 0x50},
		// iteration 4 (this batch) — all decompile-confirmed Reason-shape in
		// v83/v84/v87/v95 (Decode1(reason) -> NoticeFailReason; the GetUser*Failed
		// arms additionally re-send the transfer-field packet when reason==73, which
		// reads NO further bytes).
		{"GetSearchITCListFailed", 0x18, 0x51},    // v83 0x5a49e3 / v84 0x5b4ed3 / v87 0x5d4ad3 / v95 0x575fa0
		{"GetUserPurchaseItemFailed", 0x22, 0x49}, // v83 0x5a4c2a / v84 0x5b511a / v87 0x5d4d1a / v95 0x575fd0
		{"GetUserSaleItemFailed", 0x24, 0x49},     // v83 0x5a4ce7 / v84 0x5b51d7 / v87 0x5d4dd7 / v95 0x576000
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			input := NewMtsResultReason(c.mode, c.reason)
			expected := []byte{c.mode, c.reason}
			actual := test.Encode(t, ctx, input.Encode, nil)
			if !bytes.Equal(actual, expected) {
				t.Errorf("golden mismatch: got %v want %v", actual, expected)
			}
		})
	}
}

func TestMtsResultReasonRoundTrip(t *testing.T) {
	input := NewMtsResultReason(0x16, 0x42)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultReason{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Reason() != input.Reason() {
				t.Errorf("reason: got %v, want %v", output.Reason(), input.Reason())
			}
		})
	}
}

// MtsResultTwoInts arms — sub-handler reads exactly two Decode4 ints after the
// dispatcher mode byte. MOVE_ITC_PURCHASE_ITEM_LTOS_DONE (0x27) is the pinned
// representative; NOTIFY_CANCEL_WISH_RESULT (0x3D) shares the identical
// mode + Decode4 + Decode4 wire shape.
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultTwoInts version=gms_v83 ida=0x5a4d68
// packet-audit:verify packet=field/clientbound/FieldMtsResultTwoInts version=gms_v84 ida=0x5b5258
// packet-audit:verify packet=field/clientbound/FieldMtsResultTwoInts version=gms_v87 ida=0x5d4e58
// packet-audit:verify packet=field/clientbound/FieldMtsResultTwoInts version=gms_v95 ida=0x5760a0
func TestMtsResultTwoIntsGolden(t *testing.T) {
	// mode 0x27 = MOVE_ITC_PURCHASE_ITEM_LTOS_DONE. Sub-handler decompile
	// (v95 0x5760a0): Decode4(v3) -> SetTab(v3-1), Decode4(v4) -> SetSelectedNo(v4).
	// mode 0x3D = NOTIFY_CANCEL_WISH_RESULT (v95 0x576f00): Decode4(v3) Decode4(v4).
	// Both read exactly two big-bytes-after-mode int32s; the codec writes them
	// little-endian via WriteInt (matching the socket WriteInt contract).
	ctx := test.CreateContext("GMS", 95, 0)
	cases := []struct {
		name string
		mode byte
		a    uint32
		b    uint32
	}{
		{"MoveITCPurchaseItemLtoSDone", 0x27, 0x00000003, 0x0000000A}, // tab+1, selectedNo
		{"NotifyCancelWishResult", 0x3D, 0x00000005, 0x00000002},      // count d, count x
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			input := NewMtsResultTwoInts(c.mode, c.a, c.b)
			expected := []byte{
				c.mode,
				byte(c.a), byte(c.a >> 8), byte(c.a >> 16), byte(c.a >> 24),
				byte(c.b), byte(c.b >> 8), byte(c.b >> 16), byte(c.b >> 24),
			}
			actual := test.Encode(t, ctx, input.Encode, nil)
			if !bytes.Equal(actual, expected) {
				t.Errorf("golden mismatch: got %v want %v", actual, expected)
			}
		})
	}
}

func TestMtsResultTwoIntsRoundTrip(t *testing.T) {
	input := NewMtsResultTwoInts(0x27, 0x11223344, 0x55667788)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultTwoInts{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

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
		input := NewMtsResultRegisterSaleEntryFailed(0x42, 0) // 'B' -> NoticeFailReason only
		expected := []byte{0x1E, 0x42}
		actual := test.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("golden mismatch: got %v want %v", actual, expected)
		}
	})

	// reason == 0x48: mode + reason + Decode2 sale-limit short (little-endian).
	t.Run("SaleLimitReason", func(t *testing.T) {
		input := NewMtsResultRegisterSaleEntryFailed(0x48, 0x0064) // 'H', limit 100
		expected := []byte{0x1E, 0x48, 0x64, 0x00}
		actual := test.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("golden mismatch: got %v want %v", actual, expected)
		}
	})
}

func TestMtsResultRegisterSaleEntryFailedRoundTrip(t *testing.T) {
	for _, in := range []MtsResultRegisterSaleEntryFailed{
		NewMtsResultRegisterSaleEntryFailed(0x42, 0),
		NewMtsResultRegisterSaleEntryFailed(0x48, 0x1234),
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
		input := NewMtsResultSuccessBidInfo(1, 0, 0, [8]byte{})
		expected := []byte{0x3E, 0x01, 0x00, 0x00, 0x00, 0x00}
		actual := test.Encode(t, ctx, input.Encode, nil)
		if !bytes.Equal(actual, expected) {
			t.Errorf("golden mismatch: got %v want %v", actual, expected)
		}
	})

	// itemId > 0: mode + soldFlag + itemId + price + 8-byte FILETIME buffer.
	t.Run("WithItem", func(t *testing.T) {
		date := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
		input := NewMtsResultSuccessBidInfo(1, 0x00204FCE, 0x000F4240, date) // itemId 2117070, price 1,000,000
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
		NewMtsResultSuccessBidInfo(0, 0, 0, [8]byte{}),
		NewMtsResultSuccessBidInfo(1, 0x00204FCE, 0x000F4240, date),
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
