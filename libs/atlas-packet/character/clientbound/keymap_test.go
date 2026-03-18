package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas-packet/test"
)

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
