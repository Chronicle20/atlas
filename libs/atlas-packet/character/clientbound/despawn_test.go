package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// CharacterDespawn v48 byte-fixture — REMOVE_PLAYER_FROM_MAP, op 101 (0x65).
//
// Client read — CUserPool::OnUserLeaveField (sub_6B2976 @0x6b2976):
// Decode4(charId)@0x6b298e, then internal record removal (no further decodes).
// Single int32 body == v61/v83 (no version gate). v48 op 101.
//
// packet-audit:verify packet=character/clientbound/CharacterDespawn version=gms_v48 ida=0x6b2976
func TestCharacterDespawnV48ByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	got := CharacterDespawn{characterId: 12345}.Encode(nil, ctx)(nil)
	want := []byte{0x39, 0x30, 0x00, 0x00} // characterId 12345 (Encode4) /*0x6b298e*/
	if !bytes.Equal(got, want) {
		t.Errorf("v48 CharacterDespawn wire: got %x want %x", got, want)
	}
}

// packet-audit:verify packet=character/clientbound/CharacterDespawn version=gms_v83 ida=0x9722f9
// packet-audit:verify packet=character/clientbound/CharacterDespawn version=gms_v87 ida=0x9f727f
// packet-audit:verify packet=character/clientbound/CharacterDespawn version=gms_v95 ida=0x94d4c0
// packet-audit:verify packet=character/clientbound/CharacterDespawn version=gms_v84 ida=0x9b2299
// packet-audit:verify packet=character/clientbound/CharacterDespawn version=jms_v185 ida=0xa43fd8
func TestCharacterDespawnRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterDespawn{characterId: 12345}
			output := CharacterDespawn{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != input.CharacterId() {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), input.CharacterId())
			}
		})
	}
}
