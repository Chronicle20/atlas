package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v83 ida=0x5f9d15
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v84 ida=0x60ed4a
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v87 ida=0x6315de
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=gms_v95 ida=0x5d9e10
// packet-audit:verify packet=character/clientbound/DeleteCharacterResponse version=jms_v185 ida=0x66f9fe
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
