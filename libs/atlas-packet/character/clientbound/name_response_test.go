package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v48 ida=0x5016db
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v83 ida=0x5f9c72
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v84 ida=0x60eca7
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v87 ida=0x63153b
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=gms_v95 ida=0x5d5790
// packet-audit:verify packet=character/clientbound/CharacterNameResponse version=jms_v185 ida=0x66f957
// TestCharacterNameResponseByteOutputV48 pins the gms_v48 CHAR_NAME_RESPONSE
// (op 13). IDA: CLogin::OnCheckDuplicatedIDResult = sub_5016DB @0x5016db
// (GMS_v48_1_DEVM.exe) reads DecodeStr(name)@0x5016eb + Decode1(result)@0x501705
// — byte-identical to the v83 shape (WriteAsciiString + WriteByte). No codec gate.
func TestCharacterNameResponseByteOutputV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	m := CharacterNameResponse{name: "Test", code: 0}
	if got := pt.Encode(t, ctx, m.Encode, nil); !bytes.Equal(got, []byte{
		0x04, 0x00, 'T', 'e', 's', 't', 0x00,
	}) {
		t.Errorf("v48 name_response: got %v", got)
	}
}

func TestCharacterNameResponseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterNameResponse{name: "TestChar", code: 0}
			output := CharacterNameResponse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != input.Name() {
				t.Errorf("name: got %v, want %v", output.Name(), input.Name())
			}
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}
