package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v72 ida=0x51626c
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v79 ida=0x51d328
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v83 ida=0x531e00
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v84 ida=0x53e086
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v87 ida=0x5596b1
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v95 ida=0x535490
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=jms_v185 ida=0x56f286
// TestMultiChatByteOutputV48 pins the gms_v48 MULTICHAT (op 0x50 = 80) clientbound
// wire. IDA: CField::OnGroupMessage @0x4c6dd6 (GMS_v48_1_DEVM.exe) reads —
// Decode1(mode) @0x4c6dee → mode byte; DecodeStr(from) @0x4c6e1d → sender name;
// DecodeStr(message) @0x4c6e77 → chat message (the trailing CHATLOG_ADD /
// IsInBlackList calls are display logic, not wire reads). Read order byte-identical
// to the v61 golden (version-agnostic codec); v48 op 0x50 is Delta-20 vs v61 0x64.
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v48 ida=0x4c6dd6
func TestMultiChatByteOutputV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	input := MultiChat{mode: 1, from: "PlayerOne", message: "hi"}
	expected := []byte{
		0x01, // mode @0x4c6dee
		0x09, 0x00, 0x50, 0x6C, 0x61, 0x79, 0x65, 0x72, 0x4F, 0x6E, 0x65, // from "PlayerOne" @0x4c6e1d
		0x02, 0x00, 0x68, 0x69, // message "hi" @0x4c6e77
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v48 multichat golden mismatch: got %v want %v", actual, expected)
	}
}

// TestMultiChatByteOutputV79 pins the gms_v79 MULTICHAT (op 0x7E) clientbound
// wire. IDA: CField::OnGroupMessage @0x51d328 (GMS_v79_1_DEVM.exe) reads —
//
//	Decode1(mode)     @0x51d340 → mode byte.
//	DecodeStr(from)   @0x51d37d → sender name.
//	DecodeStr(message)@0x51d3c2 → chat message.
//
// (the trailing CHATLOG_ADD / IsInBlackList calls are display logic, not wire
// reads). WriteByte = 1 byte; WriteAsciiString = uint16-LE len + ShiftJIS bytes.
func TestMultiChatByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := MultiChat{mode: 1, from: "PlayerOne", message: "hi"}
	expected := []byte{
		0x01, // mode @0x51d340
		0x09, 0x00, 0x50, 0x6C, 0x61, 0x79, 0x65, 0x72, 0x4F, 0x6E, 0x65, // from "PlayerOne" @0x51d37d
		0x02, 0x00, 0x68, 0x69, // message "hi" @0x51d3c2
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 multichat golden mismatch: got %v want %v", actual, expected)
	}
}

// TestMultiChatByteOutputV72 pins the gms_v72 MULTICHAT (op 0x7A) clientbound
// wire. IDA: CField::OnGroupMessage @0x51626c (GMS_v72.1_U_DEVM.exe) reads —
//
//	Decode1(mode)     @0x516284 → mode byte.
//	DecodeStr(from)   @0x5162c1 → sender name.
//	DecodeStr(message)@0x516306 → chat message.
//
// (the interleaved sub_4160CB ZXString copy for the blacklist check and the
// trailing CHATLOG_ADD are display logic, not wire reads). v72 is GMS<87 so the
// body matches the v79 legacy codec byte-for-byte. WriteByte = 1 byte;
// WriteAsciiString = uint16-LE len + ASCII bytes.
func TestMultiChatByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	input := MultiChat{mode: 1, from: "PlayerOne", message: "hi"}
	expected := []byte{
		0x01, // mode @0x516284
		0x09, 0x00, 0x50, 0x6C, 0x61, 0x79, 0x65, 0x72, 0x4F, 0x6E, 0x65, // from "PlayerOne" @0x5162c1
		0x02, 0x00, 0x68, 0x69, // message "hi" @0x516306
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v72 multichat golden mismatch: got %v want %v", actual, expected)
	}
}

// TestMultiChatByteOutputV61 pins the gms_v61 MULTICHAT (op 0x64 = 100)
// clientbound wire. IDA: CField::OnGroupMessage @0x4ea7ec (GMS_v61.1_U_DEVM.exe)
// reads Decode1(mode) + DecodeStr(from) + DecodeStr(message). Body identical to
// v72 (version-agnostic codec).
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v61 ida=0x4ea7ec
func TestMultiChatByteOutputV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	input := MultiChat{mode: 1, from: "PlayerOne", message: "hi"}
	expected := []byte{
		0x01, // mode
		0x09, 0x00, 0x50, 0x6C, 0x61, 0x79, 0x65, 0x72, 0x4F, 0x6E, 0x65, // from "PlayerOne"
		0x02, 0x00, 0x68, 0x69, // message "hi"
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v61 multichat golden mismatch: got %v want %v", actual, expected)
	}
}

func TestMultiChatRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := MultiChat{mode: 1, from: "PlayerOne", message: "party chat message"}
			output := MultiChat{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.From() != input.From() {
				t.Errorf("from: got %v, want %v", output.From(), input.From())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}
