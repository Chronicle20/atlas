package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
	input := NewMtsResultGetItcListFailed(0x49) // reason 73 = the transfer-field branch value
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
	input := NewMtsResultGetSearchItcListFailed(0x51)
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
	input := NewMtsResultSaleCurrentItemToWishFailed(0x50)
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
	input := NewMtsResultGetUserPurchaseItemFailed(0x49)
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
	input := NewMtsResultGetUserSaleItemFailed(0x49)
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
	input := NewMtsResultCancelSaleItemFailed(0x42)
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
	input := NewMtsResultMoveItcPurchaseItemLtoSFailed(0x41) // reason 65 = the transfer-field re-send branch value
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
