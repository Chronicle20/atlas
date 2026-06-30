package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v79 ida=0x51d328
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v83 ida=0x531e00
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v84 ida=0x53e086
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v87 ida=0x5596b1
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=gms_v95 ida=0x535490
// packet-audit:verify packet=field/clientbound/FieldMultiChat version=jms_v185 ida=0x56f286
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
