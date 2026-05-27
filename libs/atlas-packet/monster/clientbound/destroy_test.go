package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestMonsterDestroy(t *testing.T) {
	input := NewMonsterDestroy(5001, DestroyTypeFadeOut)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMonsterDestroyBySwallow(t *testing.T) {
	input := NewMonsterDestroyBySwallow(5001, 12345)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
	// Confirm the trailing swallowCharacterId is present in the encoded bytes.
	// Wire shape for swallow: uint32(uniqueId)+byte(destroyType=4)+uint32(charId) = 9 bytes.
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 1)
	bytes := input.Encode(l, ctx)(nil)
	if len(bytes) != 9 {
		t.Errorf("swallow encode: got %d bytes, want 9 (uint32 uid + byte type + uint32 swallowCharId)", len(bytes))
	}
	// Regression check: plain destroy stays at 5 bytes.
	plain := NewMonsterDestroy(5001, DestroyTypeFadeOut)
	plainBytes := plain.Encode(l, ctx)(nil)
	if len(plainBytes) != 5 {
		t.Errorf("plain destroy encode: got %d bytes, want 5 (uint32 uid + byte type)", len(plainBytes))
	}
}
