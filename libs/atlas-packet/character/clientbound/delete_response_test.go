package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v48 ida=0x5017b6
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v83 ida=0x5f9d15
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v84 ida=0x60ed4a
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v87 ida=0x6315de
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v95 ida=0x5d9e10
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=jms_v185 ida=0x66f9fe
// TestDeleteCharacterResponseByteOutputV48 pins the gms_v48 DELETE_CHAR_RESPONSE
// (op 15). IDA: CLogin::OnDeleteCharacterResult = sub_5017B6 @0x5017b6
// (GMS_v48_1_DEVM.exe) reads Decode4(charId)@0x5017c6 + Decode1(result)@0x5017cc —
// byte-identical to the v83 shape (WriteInt + WriteByte). No codec gate.
func TestDeleteCharacterResponseByteOutputV48(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	m := DeleteCharacterResponse{characterId: 12345, code: 0}
	if got := pt.Encode(t, ctx, m.Encode, nil); !bytes.Equal(got, []byte{
		0x39, 0x30, 0x00, 0x00, // charId 12345=0x3039 (WriteInt)
		0x00, // code (WriteByte)
	}) {
		t.Errorf("v48 delete_response: got %v", got)
	}
}

func TestDeleteCharacterResponseRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := DeleteCharacterResponse{characterId: 12345, code: 0}
			output := DeleteCharacterResponse{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}
