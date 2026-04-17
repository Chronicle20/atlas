package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestDropSpawnItem(t *testing.T) {
	input := NewDropSpawn(DropEnterTypeFresh, 9001, 0, 4001000, 1234, 0, 100, -200, 5001, 80, -180, 50, false)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestDropSpawnMeso(t *testing.T) {
	input := NewDropSpawn(DropEnterTypeFresh, 9002, 500, 0, 1234, 0, 100, -200, 5001, 80, -180, 50, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestDropSpawnExisting(t *testing.T) {
	input := NewDropSpawn(DropEnterTypeExisting, 9003, 0, 4001000, 1234, 0, 100, -200, 5001, 0, 0, 0, false)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
