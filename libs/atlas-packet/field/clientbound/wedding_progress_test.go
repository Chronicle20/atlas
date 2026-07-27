package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldWeddingProgress version=gms_v72 ida=0x548c50
// packet-audit:verify packet=field/clientbound/FieldWeddingProgress version=gms_v79 ida=0x55dfbb
// packet-audit:verify packet=field/clientbound/FieldWeddingProgress version=gms_v83 ida=0x58153d
// packet-audit:verify packet=field/clientbound/FieldWeddingProgress version=gms_v84 ida=0x5911e6
// packet-audit:verify packet=field/clientbound/FieldWeddingProgress version=gms_v87 ida=0x5b012e
// packet-audit:verify packet=field/clientbound/FieldWeddingProgress version=gms_v95 ida=0x5640f0
// packet-audit:verify packet=field/clientbound/FieldWeddingProgress version=jms_v185 ida=0x5d6612
func TestWeddingProgressGolden(t *testing.T) {
	input := NewWeddingProgress(0x02, 0x01020304, 0x05060708)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x02, 0x04, 0x03, 0x02, 0x01, 0x08, 0x07, 0x06, 0x05}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

// TestWeddingProgressByteOutputV79 pins the gms_v79 WEDDING_PROGRESS clientbound
// read. IDA: CField_Wedding::OnWeddingProgress @0x55dfbb (GMS_v79_1_DEVM.exe) reads
// Decode1(step)@0x55e021 + Decode4(groomId)@0x55e026 + Decode4(brideId)@0x55e039.
func TestWeddingProgressByteOutputV79(t *testing.T) {
	input := NewWeddingProgress(0x02, 0x01020304, 0x05060708)
	ctx := test.CreateContext("GMS", 79, 1)
	expected := []byte{0x02, 0x04, 0x03, 0x02, 0x01, 0x08, 0x07, 0x06, 0x05}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestWeddingProgressByteOutputV72 pins the gms_v72 WEDDING_PROGRESS clientbound
// read. IDA: CField_Wedding::OnWeddingProgress @0x548c50 (GMS_v72.1_U_DEVM.exe)
// reads Decode1(step)@0x548cb6 + Decode4(groomId this+483)@0x548cbb +
// Decode4(brideId this+484)@0x548cce. Body identical to v79 (GMS keeps the step
// byte). This clientbound fixture completes the OnWeddingProgress worst-of family
// so the serverbound WEDDING_ACTION/WEDDING_TALK v72 cells can promote.
func TestWeddingProgressByteOutputV72(t *testing.T) {
	input := NewWeddingProgress(0x02, 0x01020304, 0x05060708)
	ctx := test.CreateContext("GMS", 72, 1)
	expected := []byte{0x02, 0x04, 0x03, 0x02, 0x01, 0x08, 0x07, 0x06, 0x05}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestWeddingProgressByteOutputV48 pins the gms_v48 WEDDING_PROGRESS clientbound
// read. IDA: CField_Wedding::OnWeddingProgress @0x4e22ff (GMS_v48_1_DEVM.exe)
// reads Decode1(mode/step) @0x4e2365 + Decode4(groomId this+113) @0x4e236a +
// Decode4(brideId this+114) @0x4e237d. Body identical to v61 (GMS keeps the step
// byte). This clientbound fixture completes the OnWeddingProgress worst-of family
// so the serverbound WEDDING_ACTION/WEDDING_TALK v48 cells can promote.
// packet-audit:verify packet=field/clientbound/FieldWeddingProgress version=gms_v48 ida=0x4e22ff
func TestWeddingProgressByteOutputV48(t *testing.T) {
	input := NewWeddingProgress(0x02, 0x01020304, 0x05060708)
	ctx := test.CreateContext("GMS", 48, 1)
	expected := []byte{0x02, 0x04, 0x03, 0x02, 0x01, 0x08, 0x07, 0x06, 0x05}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v48 golden mismatch: got %v want %v", actual, expected)
	}
}

// TestWeddingProgressByteOutputV61 pins the gms_v61 WEDDING_PROGRESS clientbound
// read. IDA: CField_Wedding::OnWeddingProgress @0x513473 (GMS_v61.1_U_DEVM.exe)
// reads Decode1(step) + Decode4(groomId this+461) + Decode4(brideId this+462).
// Body identical to v72 (GMS keeps the step byte). This clientbound fixture
// completes the OnWeddingProgress worst-of family so the serverbound
// WEDDING_ACTION/WEDDING_TALK v61 cells can promote.
// packet-audit:verify packet=field/clientbound/FieldWeddingProgress version=gms_v61 ida=0x513473
func TestWeddingProgressByteOutputV61(t *testing.T) {
	input := NewWeddingProgress(0x02, 0x01020304, 0x05060708)
	ctx := test.CreateContext("GMS", 61, 1)
	expected := []byte{0x02, 0x04, 0x03, 0x02, 0x01, 0x08, 0x07, 0x06, 0x05}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v61 golden mismatch: got %v want %v", actual, expected)
	}
}

// JMS drops the leading step byte: groomId, brideId only.
func TestWeddingProgressGoldenJMS(t *testing.T) {
	input := NewWeddingProgress(0x02, 0x01020304, 0x05060708)
	ctx := test.CreateContext("JMS", 185, 1)
	expected := []byte{0x04, 0x03, 0x02, 0x01, 0x08, 0x07, 0x06, 0x05}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("jms golden mismatch: got %v want %v", actual, expected)
	}
}

func TestWeddingProgressRoundTrip(t *testing.T) {
	input := NewWeddingProgress(0x02, 0x01020304, 0x05060708)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
