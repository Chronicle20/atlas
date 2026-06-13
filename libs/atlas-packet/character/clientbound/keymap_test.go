package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=gms_v83 ida=0x58ddb4
// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=gms_v87 ida=0x5bd279
// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=gms_v95 ida=0x568c30
// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=jms_v185 ida=0x5e79aa
// packet-audit:verify packet=character/clientbound/CharacterKeyMap version=gms_v84 ida=0x59dda7
func TestCharacterKeyMap(t *testing.T) {
	keys := map[int32]KeyBinding{
		2:  {KeyType: 4, KeyAction: 10},
		16: {KeyType: 4, KeyAction: 8},
		41: {KeyType: 4, KeyAction: 11},
	}
	input := NewCharacterKeyMap(keys)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestCharacterKeyMapResetToDefault(t *testing.T) {
	input := NewCharacterKeyMapResetToDefault()
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
