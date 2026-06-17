package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
	input := NewMtsResultMoveItcPurchaseItemLtoSDone(0x00000003, 0x0000000A)
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
	input := NewMtsResultNotifyCancelWishResult(0x00000005, 0x00000002)
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
