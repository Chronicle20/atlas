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

// MtsResultEmpty arms — sub-handler reads NOTHING after the dispatcher mode byte
// (StringPool notice only). Addresses: REGISTER_SALE_ENTRY_DONE (0x1D) used as
// the pinned representative per version; every other Empty arm shares the
// identical zero-body shape (see mts_operation_body.go for their per-version
// addresses). Only ONE verify marker per (packet,version) is permitted; the
// representative below stands for the whole Empty-shape group, and the golden
// table exercises each covered mode against the byte-proven codec.
//
// packet-audit:verify packet=field/clientbound/FieldMtsResultEmpty version=gms_v83 ida=0x5a4674
// packet-audit:verify packet=field/clientbound/FieldMtsResultEmpty version=gms_v84 ida=0x5b4b64
// packet-audit:verify packet=field/clientbound/FieldMtsResultEmpty version=gms_v87 ida=0x5d4748
// packet-audit:verify packet=field/clientbound/FieldMtsResultEmpty version=gms_v95 ida=0x575cd0
func TestMtsResultEmptyGolden(t *testing.T) {
	// mode 0x1D = REGISTER_SALE_ENTRY_DONE. Sub-handler decompile (v95 0x575cd0):
	// GetString(0x12BC) + CUtilDlg::Notice + ResetInfo — no CInPacket::Decode*
	// after the dispatcher's Decode1(mode). So the wire is exactly the mode byte.
	//
	// Each case is decompile-verified Empty-shape (StringPool::GetString +
	// CUtilDlg::Notice, zero CInPacket::Decode* after the dispatcher Decode1) in
	// ALL FOUR versions; the per-version sub-handler addresses are in
	// mts_operation_body.go. iteration 1: 0x1D/0x1F/0x29/0x2A. iteration 2:
	// 0x25/0x2B/0x2C/0x2E/0x2F/0x30. iteration 3:
	// 0x31/0x32/0x34/0x36/0x38/0x3C.
	cases := []struct {
		name string
		mode byte
	}{
		// iteration 1
		{"RegisterSaleEntryDone", 0x1D},
		{"SaleCurrentItemToWishDone", 0x1F},
		{"SetZzimDone", 0x29},
		{"SetZzimFailed", 0x2A},
		// iteration 2
		{"CancelSaleItemDone", 0x25},     // v83 0x5a4d14 / v84 0x5b5204 / v87 0x5d4e04 / v95 0x576030
		{"DeleteZzimDone", 0x2B},         // v83 0x5a4e66 / v84 0x5b5356 / v87 0x5d4f59 / v95 0x5761c0
		{"DeleteZzimFailed", 0x2C},       // v83 0x5a4e91 / v84 0x5b5381 / v87 0x5d4f84 / v95 0x5761f0
		{"LoadWishSaleListFailed", 0x2E}, // v83 0x5a4fdc / v84 0x5b54cc / v87 0x5d50cf / v95 0x576230
		{"BuyWishDone", 0x2F},            // v83 0x5a5011 / v84 0x5b5501 / v87 0x5d5104 / v95 0x576270
		{"BuyWishFailed", 0x30},          // v83 0x5a503c / v84 0x5b552c / v87 0x5d512f / v95 0x5762a0
		// iteration 3 (this batch) — all decompile-confirmed Empty-shape in
		// v83/v84/v87/v95 (StringPool::GetString + CUtilDlg::Notice + this[6]=0
		// member store; NO CInPacket::Decode* after the dispatcher Decode1).
		{"CancelWishDone", 0x31},         // v83 0x5a5071 / v84 0x5b5561 / v87 0x5d5164 / v95 0x5762e0
		{"CancelWishFailed", 0x32},       // v83 0x5a50df / v84 0x5b5596 / v87 0x5d5199 / v95 0x576320
		{"BuyItemFailed", 0x34},          // v83 0x5a513f / v84 0x5b55f6 / v87 0x5d51f9 / v95 0x576390
		{"BuyZzimItemFailed", 0x36},      // v83 0x5a519f / v84 0x5b5656 / v87 0x5d5259 / v95 0x576400
		{"RegisterWishItemFailed", 0x38}, // v83 0x5a5209 / v84 0x5b56c0 / v87 0x5d52c3 / v95 0x576480
		{"BidAuctionFailed", 0x3C},       // v83 0x5a5444 / v84 0x5b58fb / v87 0x5d54fe / v95 0x5764c0
	}
	ctx := test.CreateContext("GMS", 95, 0)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			input := NewMtsResultEmpty(c.mode)
			expected := []byte{c.mode} // dispatcher mode byte; sub-handler reads no further fields
			actual := test.Encode(t, ctx, input.Encode, nil)
			if !bytes.Equal(actual, expected) {
				t.Errorf("golden mismatch: got %v want %v", actual, expected)
			}
		})
	}
}

func TestMtsResultEmptyRoundTrip(t *testing.T) {
	input := NewMtsResultEmpty(0x29)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			output := MtsResultEmpty{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

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
